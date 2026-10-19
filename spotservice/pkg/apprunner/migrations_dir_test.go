package apprunner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveMigrationsDir_usesConfiguredDirectory(t *testing.T) {
	dir := t.TempDir()

	got, err := resolveMigrationsDir(dir)
	if err != nil {
		t.Fatalf("resolveMigrationsDir() error = %v", err)
	}
	if got != dir {
		t.Fatalf("dir = %q, want %q", got, dir)
	}
}

func TestResolveMigrationsDir_relativePathStaysRelative(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "migrations"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}
	t.Chdir(root)

	got, err := resolveMigrationsDir("migrations")
	if err != nil {
		t.Fatalf("resolveMigrationsDir() error = %v", err)
	}
	if got != "migrations" {
		t.Fatalf("dir = %q, want migrations", got)
	}
}

func TestResolveMigrationsDir_missingPathIsStartupError(t *testing.T) {
	_, err := resolveMigrationsDir(filepath.Join(t.TempDir(), "missing"))
	if err == nil {
		t.Fatal("expected error for missing migrations dir")
	}
}

func TestResolveMigrationsDir_fileIsNotDirectory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "migrations.sql")
	if err := os.WriteFile(path, []byte("select 1"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	_, err := resolveMigrationsDir(path)
	if err == nil {
		t.Fatal("expected error for a file path")
	}
}

func TestResolveMigrationsDir_doesNotSearchServiceDirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "spotservice", "migrations"), 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	t.Chdir(root)

	_, err := resolveMigrationsDir("migrations")
	if err == nil {
		t.Fatal("expected error: sibling spotservice/migrations must not be selected")
	}
}
