package envloader_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/exchange-grpc/shared/envloader"
)

func TestLoad_setsMissingVarsOnly(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	content := "PHASE3_TEST_A=from-file\nPHASE3_TEST_B=from-file\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	t.Setenv("PHASE3_TEST_A", "from-env")
	_ = os.Unsetenv("PHASE3_TEST_B")

	envloader.Load(path)

	if got := os.Getenv("PHASE3_TEST_A"); got != "from-env" {
		t.Fatalf("PHASE3_TEST_A = %q, want from-env (must not overwrite)", got)
	}
	if got := os.Getenv("PHASE3_TEST_B"); got != "from-file" {
		t.Fatalf("PHASE3_TEST_B = %q, want from-file", got)
	}
}

func TestLoad_missingFileIsOK(t *testing.T) {
	envloader.Load(filepath.Join(t.TempDir(), "no-such.env"))
}

func TestLoad_explicitEmptyEnvNotOverwritten(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env")
	if err := os.WriteFile(path, []byte("PHASE3_TEST_EMPTY=from-file\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	t.Setenv("PHASE3_TEST_EMPTY", "")
	envloader.Load(path)

	if got, ok := os.LookupEnv("PHASE3_TEST_EMPTY"); !ok || got != "" {
		t.Fatalf("PHASE3_TEST_EMPTY = %q ok=%v, want empty explicitly set env", got, ok)
	}
}
