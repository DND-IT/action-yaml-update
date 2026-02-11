// Package inputs handles parsing GitHub Action inputs from environment variables.
package inputs

import (
	"fmt"
	"os"
	"strings"
)

// Config holds all parsed input values.
type Config struct {
	Files          []string
	Mode           string
	Keys           []string
	Values         []string
	ImageName      string
	ImageTag       string
	CreatePR       bool
	TargetBranch   string
	PRBranch       string
	PRTitle        string
	PRBody         string
	PRLabels       []string
	PRReviewers    []string
	CommitMessage  string
	Token          string
	AutoMerge      bool
	MergeMethod    string
	DryRun         bool
	GitUserName    string
	GitUserEmail   string
	GithubRepo     string
	GithubServerURL string
	GithubAPIURL   string
	GithubGraphQLURL string
}

// Parse reads and validates inputs from environment variables.
func Parse() (*Config, error) {
	cfg := &Config{
		Mode:           getEnv("MODE", "key"),
		CreatePR:       parseBool(getEnv("CREATE_PR", "true")),
		TargetBranch:   getEnv("TARGET_BRANCH", ""),
		PRBranch:       getEnv("PR_BRANCH", ""),
		PRTitle:        getEnv("PR_TITLE", "chore: update YAML values"),
		PRBody:         getEnv("PR_BODY", ""),
		CommitMessage:  getEnv("COMMIT_MESSAGE", "chore: update YAML values"),
		Token:          getEnvFallback("TOKEN", "GITHUB_TOKEN", ""),
		AutoMerge:      parseBool(getEnv("AUTO_MERGE", "false")),
		MergeMethod:    getEnv("MERGE_METHOD", "SQUASH"),
		DryRun:         parseBool(getEnv("DRY_RUN", "false")),
		GitUserName:    getEnv("GIT_USER_NAME", "github-actions[bot]"),
		GitUserEmail:   getEnv("GIT_USER_EMAIL", "41898282+github-actions[bot]@users.noreply.github.com"),
		GithubRepo:     os.Getenv("GITHUB_REPOSITORY"),
		GithubServerURL: getEnvDefault("GITHUB_SERVER_URL", "https://github.com"),
		GithubAPIURL:   getEnvDefault("GITHUB_API_URL", "https://api.github.com"),
		GithubGraphQLURL: getEnvDefault("GITHUB_GRAPHQL_URL", "https://api.github.com/graphql"),
	}

	// Parse files
	cfg.Files = parseList(getEnv("FILES", ""), "\n")
	if len(cfg.Files) == 0 {
		return nil, fmt.Errorf("'files' input is required")
	}

	// Validate mode
	if cfg.Mode != "key" && cfg.Mode != "image" {
		return nil, fmt.Errorf("invalid mode '%s'. Must be 'key' or 'image'", cfg.Mode)
	}

	// Parse mode-specific inputs
	if cfg.Mode == "key" {
		cfg.Keys = parseList(getEnv("KEYS", ""), "\n")
		cfg.Values = parseList(getEnv("VALUES", ""), "\n")

		if len(cfg.Keys) == 0 {
			return nil, fmt.Errorf("'keys' input is required for mode=key")
		}
		if len(cfg.Values) == 0 {
			return nil, fmt.Errorf("'values' input is required for mode=key")
		}
		if len(cfg.Keys) != len(cfg.Values) {
			return nil, fmt.Errorf("number of keys (%d) must match number of values (%d)", len(cfg.Keys), len(cfg.Values))
		}
	} else {
		cfg.ImageName = getEnv("IMAGE_NAME", "")
		cfg.ImageTag = getEnv("IMAGE_TAG", "")

		if cfg.ImageName == "" {
			return nil, fmt.Errorf("'image_name' input is required for mode=image")
		}
		if cfg.ImageTag == "" {
			return nil, fmt.Errorf("'image_tag' input is required for mode=image")
		}
	}

	// Parse optional lists
	cfg.PRLabels = parseList(getEnv("PR_LABELS", ""), ",")
	cfg.PRReviewers = parseList(getEnv("PR_REVIEWERS", ""), ",")

	return cfg, nil
}

func getEnv(name, defaultValue string) string {
	key := "INPUT_" + strings.ToUpper(strings.ReplaceAll(name, "-", "_"))
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}

func getEnvFallback(name, fallback, defaultValue string) string {
	if v := getEnv(name, ""); v != "" {
		return v
	}
	if v := os.Getenv(fallback); v != "" {
		return v
	}
	return defaultValue
}

func getEnvDefault(name, defaultValue string) string {
	if v := os.Getenv(name); v != "" {
		return v
	}
	return defaultValue
}

func parseBool(s string) bool {
	lower := strings.ToLower(strings.TrimSpace(s))
	return lower == "true" || lower == "yes" || lower == "1"
}

func parseList(s, sep string) []string {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var result []string
	for _, item := range strings.Split(s, sep) {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}
