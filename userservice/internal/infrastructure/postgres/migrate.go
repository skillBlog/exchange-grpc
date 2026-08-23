package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
)

// RunMigrations применяет SQL-миграции goose через уже открытый pgxpool.
func RunMigrations(ctx context.Context, db *DB, migrationsDir string) error {
	if db == nil || db.Pool == nil {
		return fmt.Errorf("database is not configured")
	}

	sqlDB := stdlib.OpenDBFromPool(db.Pool)
	defer sqlDB.Close()

	if err := sqlDB.PingContext(ctx); err != nil {
		return fmt.Errorf("ping database: %w", err)
	}

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set goose dialect: %w", err)
	}

	if err := goose.UpContext(ctx, sqlDB, migrationsDir); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

// Ping проверяет доступность PostgreSQL.
func Ping(ctx context.Context, databaseURL string) error {
	sqlDB, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	return sqlDB.PingContext(ctx)
}
