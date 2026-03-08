package internal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewConfigValid(t *testing.T) {
	dir := t.TempDir()
	cfg, err := NewConfig(dir, "go build .", "./app")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Root != dir {
		t.Errorf("expected Root=%q, got %q", dir, cfg.Root)
	}
	if cfg.BuildCmd != "go build ." {
		t.Errorf("expected BuildCmd=%q, got %q", "go build .", cfg.BuildCmd)
	}
	if cfg.ExecCmd != "./app" {
		t.Errorf("expected ExecCmd=%q, got %q", "./app", cfg.ExecCmd)
	}
}

func TestNewConfigEmptyRoot(t *testing.T) {
	_, err := NewConfig("", "go build .", "./app")
	if err == nil {
		t.Fatal("expected error for empty root")
	}
}

func TestNewConfigNonexistentRoot(t *testing.T) {
	_, err := NewConfig("/nonexistent/path/xyz", "go build .", "./app")
	if err == nil {
		t.Fatal("expected error for nonexistent root")
	}
}

func TestNewConfigRootIsFile(t *testing.T) {
	dir := t.TempDir()
	f := filepath.Join(dir, "file.txt")
	os.WriteFile(f, []byte("hi"), 0644)

	_, err := NewConfig(f, "go build .", "./app")
	// Root exists but is not a directory — config should still accept it
	// since current validation only checks existence. If this ever changes,
	// update this test accordingly.
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewConfigEmptyBuildCmd(t *testing.T) {
	dir := t.TempDir()
	_, err := NewConfig(dir, "", "./app")
	if err == nil {
		t.Fatal("expected error for empty build command")
	}
}

func TestNewConfigEmptyExecCmd(t *testing.T) {
	dir := t.TempDir()
	_, err := NewConfig(dir, "go build .", "")
	if err == nil {
		t.Fatal("expected error for empty exec command")
	}
}
