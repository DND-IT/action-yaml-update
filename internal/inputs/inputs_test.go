package inputs

import (
	"os"
	"path/filepath"
	"testing"
)

// helper to set INPUT_ env vars and clean up after test.
func setEnv(t *testing.T, key, value string) {
	t.Helper()
	envKey := "INPUT_" + key
	t.Setenv(envKey, value)
}

func TestParseRequiresFilesOrFilesFrom(t *testing.T) {
	// Set minimum required inputs for key mode but no files
	setEnv(t, "MODE", "key")
	setEnv(t, "KEYS", "app.version")
	setEnv(t, "VALUES", "1.0.0")

	_, err := Parse()
	if err == nil {
		t.Fatal("expected error when neither files nor files_from is set")
	}
	if got := err.Error(); got != "no files to process: set 'files' and/or 'files_from'" {
		t.Fatalf("unexpected error: %s", got)
	}
}

func TestParseFilesFromDiscovery(t *testing.T) {
	dir := t.TempDir()
	// Create nested structure with YAML files
	mkFile(t, filepath.Join(dir, "dev", "values.yaml"))
	mkFile(t, filepath.Join(dir, "staging", "values.yaml"))
	mkFile(t, filepath.Join(dir, "prod", "values.yaml"))
	mkFile(t, filepath.Join(dir, "dev", "secrets.yaml"))
	mkFile(t, filepath.Join(dir, "readme.md")) // not YAML

	setEnv(t, "FILES_FROM", dir)
	setEnv(t, "MODE", "key")
	setEnv(t, "KEYS", "app.version")
	setEnv(t, "VALUES", "1.0.0")

	cfg, err := Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find 4 YAML files (3 values.yaml + 1 secrets.yaml), not readme.md
	if len(cfg.Files) != 4 {
		t.Fatalf("expected 4 files, got %d: %v", len(cfg.Files), cfg.Files)
	}

	// Should be sorted
	for i := 1; i < len(cfg.Files); i++ {
		if cfg.Files[i] < cfg.Files[i-1] {
			t.Fatalf("files not sorted: %v", cfg.Files)
		}
	}
}

func TestParseFilesFromWithFilter(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "dev", "values.yaml"))
	mkFile(t, filepath.Join(dir, "staging", "values.yaml"))
	mkFile(t, filepath.Join(dir, "dev", "secrets.yaml"))
	mkFile(t, filepath.Join(dir, "dev", "config.yml"))

	setEnv(t, "FILES_FROM", dir)
	setEnv(t, "FILES_FILTER", "values.yaml")
	setEnv(t, "MODE", "key")
	setEnv(t, "KEYS", "app.version")
	setEnv(t, "VALUES", "1.0.0")

	cfg, err := Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(cfg.Files), cfg.Files)
	}
	for _, f := range cfg.Files {
		if filepath.Base(f) != "values.yaml" {
			t.Fatalf("unexpected file: %s", f)
		}
	}
}

func TestParseFilesAndFilesFromMerged(t *testing.T) {
	dir := t.TempDir()
	explicit := filepath.Join(dir, "explicit", "values.yaml")
	mkFile(t, explicit)
	mkFile(t, filepath.Join(dir, "discover", "values.yaml"))

	setEnv(t, "FILES", explicit)
	setEnv(t, "FILES_FROM", filepath.Join(dir, "discover"))
	setEnv(t, "MODE", "key")
	setEnv(t, "KEYS", "app.version")
	setEnv(t, "VALUES", "1.0.0")

	cfg, err := Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(cfg.Files), cfg.Files)
	}
}

func TestParseFilesAndFilesFromDedup(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "values.yaml")
	mkFile(t, f)

	// Same file in both explicit and discovery
	setEnv(t, "FILES", f)
	setEnv(t, "FILES_FROM", dir)
	setEnv(t, "MODE", "key")
	setEnv(t, "KEYS", "app.version")
	setEnv(t, "VALUES", "1.0.0")

	cfg, err := Parse()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Files) != 1 {
		t.Fatalf("expected 1 file (deduped), got %d: %v", len(cfg.Files), cfg.Files)
	}
}

func TestParseFilesFromNonExistentDir(t *testing.T) {
	setEnv(t, "FILES_FROM", "/nonexistent/path")
	setEnv(t, "MODE", "key")
	setEnv(t, "KEYS", "app.version")
	setEnv(t, "VALUES", "1.0.0")

	_, err := Parse()
	if err == nil {
		t.Fatal("expected error for non-existent directory")
	}
}

func TestParseFilesFromNotADirectory(t *testing.T) {
	f := filepath.Join(t.TempDir(), "file.yaml")
	mkFile(t, f)

	setEnv(t, "FILES_FROM", f)
	setEnv(t, "MODE", "key")
	setEnv(t, "KEYS", "app.version")
	setEnv(t, "VALUES", "1.0.0")

	_, err := Parse()
	if err == nil {
		t.Fatal("expected error when files_from points to a file")
	}
}

func TestParseFilesFromEmptyDir(t *testing.T) {
	dir := t.TempDir()

	setEnv(t, "FILES_FROM", dir)
	setEnv(t, "MODE", "key")
	setEnv(t, "KEYS", "app.version")
	setEnv(t, "VALUES", "1.0.0")

	_, err := Parse()
	if err == nil {
		t.Fatal("expected error when files_from directory has no YAML files")
	}
}

func TestDiscoverFilesYmlExtension(t *testing.T) {
	dir := t.TempDir()
	mkFile(t, filepath.Join(dir, "config.yml"))
	mkFile(t, filepath.Join(dir, "values.yaml"))

	files, err := discoverFiles(dir, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d: %v", len(files), files)
	}
}

// mkFile creates a file with parent directories and minimal YAML content.
func mkFile(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("key: value\n"), 0644); err != nil {
		t.Fatal(err)
	}
}
