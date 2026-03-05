// Package inputs handles parsing GitHub Action inputs from environment variables.
package inputs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Config holds all parsed input values.
type Config struct {
	Files          []string
	FilesFrom      string
	FilesFilter    string
	Mode           string
	Keys           []string
	Values         []string
	Value          string
	Marker         string
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
	cfg.FilesFrom = getEnv("FILES_FROM", "")
	cfg.FilesFilter = getEnv("FILES_FILTER", "")

	if cfg.FilesFrom != "" {
		discovered, err := discoverFiles(cfg.FilesFrom, cfg.FilesFilter)
		if err != nil {
			return nil, fmt.Errorf("files_from discovery: %w", err)
		}
		cfg.Files = mergeFiles(cfg.Files, discovered)
	}

	if len(cfg.Files) == 0 {
		return nil, fmt.Errorf("no files to process: set 'files' and/or 'files_from'")
	}

	// Validate mode
	if cfg.Mode != "key" && cfg.Mode != "image" && cfg.Mode != "marker" {
		return nil, fmt.Errorf("invalid mode '%s'. Must be 'key', 'image', or 'marker'", cfg.Mode)
	}

	// Parse shared value input
	cfg.Value = getEnv("VALUE", "")

	// Parse mode-specific inputs
	switch cfg.Mode {
	case "key":
		cfg.Keys = parseList(getEnv("KEYS", ""), "\n")
		cfg.Values = parseList(getEnv("VALUES", ""), "\n")

		if len(cfg.Keys) == 0 {
			return nil, fmt.Errorf("'keys' input is required for mode=key")
		}

		// If singular value is set, expand it to all keys
		if cfg.Value != "" && len(cfg.Values) == 0 {
			cfg.Values = make([]string, len(cfg.Keys))
			for i := range cfg.Values {
				cfg.Values[i] = cfg.Value
			}
		}

		if len(cfg.Values) == 0 {
			return nil, fmt.Errorf("'values' or 'value' input is required for mode=key")
		}
		if len(cfg.Keys) != len(cfg.Values) {
			return nil, fmt.Errorf("number of keys (%d) must match number of values (%d)", len(cfg.Keys), len(cfg.Values))
		}

	case "image":
		cfg.ImageName = getEnv("IMAGE_NAME", "")
		cfg.ImageTag = getEnv("IMAGE_TAG", "")

		if cfg.ImageName == "" {
			return nil, fmt.Errorf("'image_name' input is required for mode=image")
		}
		if cfg.ImageTag == "" {
			return nil, fmt.Errorf("'image_tag' input is required for mode=image")
		}

	case "marker":
		cfg.Marker = getEnv("MARKER", "x-yaml-update")

		if cfg.Value == "" {
			return nil, fmt.Errorf("'value' input is required for mode=marker")
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

// discoverFiles walks dir recursively and returns sorted paths to YAML files.
// If filter is non-empty, only files whose base name matches filter are included.
func discoverFiles(dir, filter string) ([]string, error) {
	info, err := os.Stat(dir)
	if err != nil {
		return nil, fmt.Errorf("directory not found: %s", dir)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", dir)
	}

	var files []string
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".yml" && ext != ".yaml" {
			return nil
		}
		if filter != "" && d.Name() != filter {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(files)
	return files, nil
}

// mergeFiles combines two file lists, removing duplicates while preserving order.
func mergeFiles(a, b []string) []string {
	seen := make(map[string]bool, len(a)+len(b))
	var result []string
	for _, f := range a {
		if !seen[f] {
			seen[f] = true
			result = append(result, f)
		}
	}
	for _, f := range b {
		if !seen[f] {
			seen[f] = true
			result = append(result, f)
		}
	}
	return result
}
