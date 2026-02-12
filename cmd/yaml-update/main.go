// Command yaml-update is the main entrypoint for the GitHub Action.
package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dnd-it/action-yaml-update/internal/github"
	"github.com/dnd-it/action-yaml-update/internal/gitops"
	"github.com/dnd-it/action-yaml-update/internal/inputs"
	"github.com/dnd-it/action-yaml-update/internal/outputs"
	"github.com/dnd-it/action-yaml-update/internal/updater"
)

func main() {
	if err := run(); err != nil {
		outputs.LogError(err.Error())
		os.Exit(1)
	}
}

func run() error {
	ctx := context.Background()

	// Parse inputs
	cfg, err := inputs.Parse()
	if err != nil {
		return err
	}

	outputs.LogInfo(fmt.Sprintf("Mode: %s", cfg.Mode))
	outputs.LogInfo(fmt.Sprintf("Files: %s", strings.Join(cfg.Files, ", ")))
	if cfg.DryRun {
		outputs.LogInfo("Dry run mode enabled — no changes will be persisted")
	}

	// Process each file
	var allChanges []updater.Change
	var changedFiles []string
	var allDiffs []string

	for _, filePath := range cfg.Files {
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			return fmt.Errorf("file not found: %s", filePath)
		}

		outputs.LogGroup(fmt.Sprintf("Processing %s", filePath))

		originalContent, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("read file %s: %w", filePath, err)
		}

		doc, err := updater.LoadYAML(originalContent)
		if err != nil {
			return fmt.Errorf("parse yaml %s: %w", filePath, err)
		}

		if doc == nil || doc.Root == nil {
			outputs.LogWarning(fmt.Sprintf("Skipping empty YAML file: %s", filePath))
			outputs.LogEndGroup()
			continue
		}

		var changes []updater.Change
		if cfg.Mode == "key" {
			changes, err = updater.UpdateKeys(doc, cfg.Keys, cfg.Values)
			if err != nil {
				outputs.LogEndGroup()
				return fmt.Errorf("update failed for %s: %w", filePath, err)
			}
		} else {
			changes = updater.UpdateImageTags(doc, cfg.ImageName, cfg.ImageTag)
		}

		if len(changes) > 0 {
			for _, c := range changes {
				outputs.LogInfo(fmt.Sprintf("  %s: %v -> %v", c.Key, c.Old, c.New))
			}
			allChanges = append(allChanges, changes...)
			changedFiles = append(changedFiles, filePath)

			newContent, err := updater.DumpYAML(doc)
			if err != nil {
				outputs.LogEndGroup()
				return fmt.Errorf("dump yaml %s: %w", filePath, err)
			}

			fileDiff := updater.Diff(filePath, originalContent, newContent)
			if fileDiff != "" {
				allDiffs = append(allDiffs, fileDiff)
			}

			if !cfg.DryRun {
				if err := os.WriteFile(filePath, newContent, 0644); err != nil {
					outputs.LogEndGroup()
					return fmt.Errorf("write file %s: %w", filePath, err)
				}
			}
		} else {
			outputs.LogInfo(fmt.Sprintf("  No changes needed for %s", filePath))
		}

		outputs.LogEndGroup()
	}

	// Write outputs
	hasChanges := len(changedFiles) > 0
	outputs.SetOutput("changed", fmt.Sprintf("%t", hasChanges))
	outputs.SetOutput("changed_files", strings.Join(changedFiles, "\n"))
	diffText := strings.Join(allDiffs, "\n")
	outputs.SetOutput("diff", diffText)

	if !hasChanges {
		outputs.LogInfo("No changes detected across any files")
		outputs.SetOutput("pr_number", "")
		outputs.SetOutput("pr_url", "")
		outputs.SetOutput("commit_sha", "")
		return nil
	}

	if cfg.DryRun {
		outputs.LogInfo("Dry run complete. Changes that would be made:")
		if diffText != "" {
			outputs.LogInfo(diffText)
		}
		outputs.SetOutput("pr_number", "")
		outputs.SetOutput("pr_url", "")
		outputs.SetOutput("commit_sha", "")
		return nil
	}

	// Git operations
	outputs.LogGroup("Git operations")

	var owner, repo string
	if cfg.GithubRepo != "" {
		parts := strings.SplitN(cfg.GithubRepo, "/", 2)
		if len(parts) == 2 {
			owner, repo = parts[0], parts[1]
		}
	}

	if err := gitops.Configure(cfg.GitUserName, cfg.GitUserEmail, cfg.Token, cfg.GithubRepo, cfg.GithubServerURL); err != nil {
		outputs.LogEndGroup()
		return fmt.Errorf("git configure: %w", err)
	}

	targetBranch := cfg.TargetBranch
	if targetBranch == "" {
		targetBranch = gitops.GetDefaultBranch()
	}

	var commitBranch string
	if cfg.CreatePR {
		prBranch := cfg.PRBranch
		if prBranch == "" {
			prBranch = generateBranchName(cfg)
		}
		if err := gitops.CreateBranch(prBranch, targetBranch); err != nil {
			outputs.LogEndGroup()
			return fmt.Errorf("create branch: %w", err)
		}
		commitBranch = prBranch
	} else {
		commitBranch = targetBranch
	}

	sha, err := gitops.CommitAndPush(changedFiles, cfg.CommitMessage, commitBranch)
	if err != nil {
		outputs.LogEndGroup()
		return fmt.Errorf("commit and push: %w", err)
	}
	outputs.SetOutput("commit_sha", sha)
	outputs.LogInfo(fmt.Sprintf("Committed and pushed: %s", sha))
	outputs.LogEndGroup()

	// Create PR
	if cfg.CreatePR {
		outputs.LogGroup("Pull request")

		prBody := cfg.PRBody
		if prBody == "" {
			var lines []string
			for _, c := range allChanges {
				lines = append(lines, fmt.Sprintf("- `%s`: `%v` → `%v`", c.Key, c.Old, c.New))
			}
			prBody = "## Changes\n\n" + strings.Join(lines, "\n")
		}

		prData, err := github.CreatePullRequest(ctx, cfg.GithubAPIURL, cfg.Token, owner, repo, cfg.PRTitle, prBody, commitBranch, targetBranch)
		if err != nil {
			outputs.LogEndGroup()
			return fmt.Errorf("create pull request: %w", err)
		}

		outputs.SetOutput("pr_number", fmt.Sprintf("%d", prData.Number))
		outputs.SetOutput("pr_url", prData.HTMLURL)
		outputs.LogInfo(fmt.Sprintf("Created PR #%d: %s", prData.Number, prData.HTMLURL))

		if len(cfg.PRLabels) > 0 {
			if err := github.AddLabels(ctx, cfg.GithubAPIURL, cfg.Token, owner, repo, prData.Number, cfg.PRLabels); err != nil {
				outputs.LogWarning(fmt.Sprintf("Failed to add labels: %v", err))
			} else {
				outputs.LogInfo(fmt.Sprintf("Added labels: %s", strings.Join(cfg.PRLabels, ", ")))
			}
		}

		if len(cfg.PRReviewers) > 0 {
			if err := github.RequestReviewers(ctx, cfg.GithubAPIURL, cfg.Token, owner, repo, prData.Number, cfg.PRReviewers); err != nil {
				outputs.LogWarning(fmt.Sprintf("Failed to request reviewers: %v", err))
			} else {
				outputs.LogInfo(fmt.Sprintf("Requested reviewers: %s", strings.Join(cfg.PRReviewers, ", ")))
			}
		}

		if cfg.AutoMerge && prData.NodeID != "" {
			if err := github.EnableAutoMerge(ctx, cfg.GithubGraphQLURL, cfg.Token, prData.NodeID, cfg.MergeMethod); err != nil {
				outputs.LogWarning(fmt.Sprintf("Failed to enable auto-merge: %v", err))
			} else {
				outputs.LogInfo(fmt.Sprintf("Enabled auto-merge (%s)", cfg.MergeMethod))
			}
		}

		outputs.LogEndGroup()
	} else {
		outputs.SetOutput("pr_number", "")
		outputs.SetOutput("pr_url", "")
	}

	outputs.LogInfo("Done!")
	return nil
}

func generateBranchName(cfg *inputs.Config) string {
	seed := fmt.Sprintf("%v%v%v%s%s", cfg.Files, cfg.Keys, cfg.Values, cfg.ImageName, cfg.ImageTag)
	hash := sha256.Sum256([]byte(seed))
	return fmt.Sprintf("yaml-update/%x-%d", hash[:4], time.Now().Unix())
}
