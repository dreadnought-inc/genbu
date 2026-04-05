package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFile_basic(t *testing.T) {
	// Find testdata relative to repo root
	cfg, err := LoadFile(filepath.Join("..", "..", "testdata", "basic.yaml"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(cfg.Variables) != 2 {
		t.Fatalf("variables count = %d, want 2", len(cfg.Variables))
	}
}

func TestLoadFile_notFound(t *testing.T) {
	_, err := LoadFile("/nonexistent/path.yaml")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadFile_invalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	if err := os.WriteFile(path, []byte(":::invalid"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadFile(path)
	if err == nil {
		t.Fatal("expected error for invalid YAML")
	}
}

func TestLoadFile_duplicateVariable(t *testing.T) {
	_, err := LoadFile(filepath.Join("..", "..", "testdata", "invalid.yaml"))
	if err == nil {
		t.Fatal("expected error for duplicate variable")
	}
}
