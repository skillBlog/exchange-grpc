package postgres

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMigrations_requiresPool(t *testing.T) {
	if err := RunMigrations(context.Background(), nil, "."); err == nil {
		t.Fatal("expected error for nil db")
	}
	if err := RunMigrations(context.Background(), &DB{}, "."); err == nil {
		t.Fatal("expected error for nil pool")
	}
}

func TestMigration00003_doesNotAlterNumericTypes(t *testing.T) {
	body := stripSQLComments(readMigration(t, "00003_orders_numeric_and_idempotency_reserve.sql"))
	if strings.Contains(body, "ALTER COLUMN") && strings.Contains(strings.ToUpper(body), "TYPE NUMERIC") {
		t.Fatal("00003 must not ALTER COLUMN ... TYPE NUMERIC: exclusive lock on orders for a no-op")
	}
}

func readMigration(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "..", "migrations", name)
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(raw)
}

func stripSQLComments(body string) string {
	var b strings.Builder
	for _, line := range strings.Split(body, "\n") {
		if i := strings.Index(line, "--"); i >= 0 {
			line = line[:i]
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
