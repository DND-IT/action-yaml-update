import { existsSync, readFileSync, writeFileSync } from "node:fs";
import { createHash } from "node:crypto";
import { parseInputs } from "./inputs.js";
import {
  updateKeys,
  updateImageTags,
  loadYaml,
  dumpYaml,
  diffYaml,
} from "./updater.js";
import * as outputs from "./outputs.js";
import * as gitOps from "./git-ops.js";
import * as githubApi from "./github-api.js";

function generateBranchName(inputs) {
  const seed = `${inputs.files}${inputs.keys}${inputs.values}${inputs.imageName}${inputs.imageTag}`;
  const hash = createHash("sha256").update(seed).digest("hex").slice(0, 8);
  return `yaml-update/${hash}-${Date.now()}`;
}

async function main() {
  // --- Parse inputs ---
  let inputs;
  try {
    inputs = parseInputs();
  } catch (e) {
    outputs.logError(e.message);
    process.exit(1);
  }

  outputs.logInfo(`Mode: ${inputs.mode}`);
  outputs.logInfo(`Files: ${inputs.files.join(", ")}`);
  if (inputs.dryRun) {
    outputs.logInfo("Dry run mode enabled — no changes will be persisted");
  }

  // --- Process each file ---
  const allChanges = [];
  const changedFiles = [];
  const allDiffs = [];

  for (const filePath of inputs.files) {
    if (!existsSync(filePath)) {
      outputs.logError(`File not found: ${filePath}`);
      process.exit(1);
    }

    outputs.logGroup(`Processing ${filePath}`);

    const originalContent = readFileSync(filePath, "utf8");
    const doc = loadYaml(originalContent);

    if (doc.contents === null) {
      outputs.logWarning(`Skipping empty YAML file: ${filePath}`);
      outputs.logEndgroup();
      continue;
    }

    let changes;
    try {
      if (inputs.mode === "key") {
        changes = updateKeys(doc, inputs.keys, inputs.values);
      } else {
        changes = updateImageTags(doc, inputs.imageName, inputs.imageTag);
      }
    } catch (e) {
      outputs.logError(`Update failed for ${filePath}: ${e.message}`);
      outputs.logEndgroup();
      process.exit(1);
    }

    if (changes.length > 0) {
      for (const c of changes) {
        outputs.logInfo(`  ${c.key}: ${c.old} -> ${c.new}`);
      }
      allChanges.push(...changes);
      changedFiles.push(filePath);

      const fileDiff = diffYaml(filePath, originalContent, doc);
      if (fileDiff) {
        allDiffs.push(fileDiff);
      }

      if (!inputs.dryRun) {
        writeFileSync(filePath, dumpYaml(doc));
      }
    } else {
      outputs.logInfo(`  No changes needed for ${filePath}`);
    }

    outputs.logEndgroup();
  }

  // --- Write outputs ---
  const hasChanges = changedFiles.length > 0;
  outputs.setOutput("changed", String(hasChanges));
  outputs.setOutput("changed_files", changedFiles.join("\n"));
  const diffText = allDiffs.join("\n");
  outputs.setOutput("diff", diffText);

  if (!hasChanges) {
    outputs.logInfo("No changes detected across any files");
    outputs.setOutput("pr_number", "");
    outputs.setOutput("pr_url", "");
    outputs.setOutput("commit_sha", "");
    process.exit(0);
  }

  if (inputs.dryRun) {
    outputs.logInfo("Dry run complete. Changes that would be made:");
    if (diffText) {
      outputs.logInfo(diffText);
    }
    outputs.setOutput("pr_number", "");
    outputs.setOutput("pr_url", "");
    outputs.setOutput("commit_sha", "");
    process.exit(0);
  }

  // --- Git operations ---
  outputs.logGroup("Git operations");

  let owner = "";
  let repo = "";
  if (inputs.githubRepository) {
    const parts = inputs.githubRepository.split("/");
    if (parts.length === 2) {
      [owner, repo] = parts;
    }
  }

  gitOps.configure(
    inputs.gitUserName,
    inputs.gitUserEmail,
    inputs.token,
    inputs.githubRepository,
    inputs.githubServerUrl,
  );

  const targetBranch = inputs.targetBranch || gitOps.getDefaultBranch();

  let commitBranch;
  if (inputs.createPr) {
    const prBranch = inputs.prBranch || generateBranchName(inputs);
    gitOps.createBranch(prBranch, targetBranch);
    commitBranch = prBranch;
  } else {
    commitBranch = targetBranch;
  }

  const sha = gitOps.commitAndPush(
    changedFiles,
    inputs.commitMessage,
    commitBranch,
  );
  outputs.setOutput("commit_sha", sha);
  outputs.logInfo(`Committed and pushed: ${sha}`);
  outputs.logEndgroup();

  // --- Create PR ---
  if (inputs.createPr) {
    outputs.logGroup("Pull request");

    let prBody = inputs.prBody;
    if (!prBody) {
      const changeLines = allChanges.map(
        (c) => `- \`${c.key}\`: \`${c.old}\` → \`${c.new}\``,
      );
      prBody = "## Changes\n\n" + changeLines.join("\n");
    }

    try {
      const prData = await githubApi.createPullRequest(
        inputs.githubApiUrl,
        inputs.token,
        owner,
        repo,
        inputs.prTitle,
        prBody,
        commitBranch,
        targetBranch,
      );

      const prNumber = prData.number;
      const prUrl = prData.html_url;
      const prNodeId = prData.node_id || "";

      outputs.setOutput("pr_number", String(prNumber));
      outputs.setOutput("pr_url", prUrl);
      outputs.logInfo(`Created PR #${prNumber}: ${prUrl}`);

      if (inputs.prLabels.length > 0) {
        await githubApi.addLabels(
          inputs.githubApiUrl,
          inputs.token,
          owner,
          repo,
          prNumber,
          inputs.prLabels,
        );
        outputs.logInfo(`Added labels: ${inputs.prLabels.join(", ")}`);
      }

      if (inputs.prReviewers.length > 0) {
        await githubApi.requestReviewers(
          inputs.githubApiUrl,
          inputs.token,
          owner,
          repo,
          prNumber,
          inputs.prReviewers,
        );
        outputs.logInfo(
          `Requested reviewers: ${inputs.prReviewers.join(", ")}`,
        );
      }

      if (inputs.autoMerge && prNodeId) {
        await githubApi.enableAutoMerge(
          inputs.githubGraphqlUrl,
          inputs.token,
          prNodeId,
          inputs.mergeMethod,
        );
        outputs.logInfo(`Enabled auto-merge (${inputs.mergeMethod})`);
      }
    } catch (e) {
      outputs.logError(`GitHub API error: ${e.message}`);
      outputs.logEndgroup();
      process.exit(1);
    }

    outputs.logEndgroup();
  } else {
    outputs.setOutput("pr_number", "");
    outputs.setOutput("pr_url", "");
  }

  outputs.logInfo("Done!");
}

main().catch((e) => {
  outputs.logError(`Unexpected error: ${e.message}`);
  process.exit(1);
});
