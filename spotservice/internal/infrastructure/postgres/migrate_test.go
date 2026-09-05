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

func TestMigration00003_doesNotAlterMarketIDType(t *testing.T) {
	body := stripSQLComments(readMigration(t, "00003_market_id_varchar.sql"))
	upper := strings.ToUpper(body)
	if strings.Contains(upper, "ALTER COLUMN") && strings.Contains(upper, "TYPE VARCHAR") {
		t.Fatal("00003 must not ALTER COLUMN id TYPE VARCHAR: exclusive lock on markets for a no-op")
	}
}

func TestMigration00005_addsLimitColumns(t *testing.T) {
	body := stripSQLComments(readMigration(t, "00005_market_limits.sql"))
	upper := strings.ToUpper(body)
	if !strings.Contains(upper, "ADD COLUMN IF NOT EXISTS MIN_ORDER_SIZE") {
		t.Fatal("00005 must ADD COLUMN IF NOT EXISTS min_order_size")
	}
	if !strings.Contains(upper, "ADD COLUMN IF NOT EXISTS QUANTITY_PRECISION") {
		t.Fatal("00005 must ADD COLUMN IF NOT EXISTS quantity_precision")
	}
	if !strings.Contains(upper, "ADD COLUMN IF NOT EXISTS MIN_NOTIONAL") {
		t.Fatal("00005 must ADD COLUMN IF NOT EXISTS min_notional")
	}
	if strings.Contains(upper, "CREATE TABLE") {
		t.Fatal("00005 must not rewrite markets table")
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
