package postgres

import (
	"context"
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
