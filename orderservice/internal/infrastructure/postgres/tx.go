package postgres

import (
	"context"
	"fmt"

	"github.com/exchange-grpc/orderservice/internal/application"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type txCtxKey struct{}

// ContextWithTx кладёт pgx.Tx в ctx. Только infrastructure: application этот тип не видит.
func ContextWithTx(ctx context.Context, tx pgx.Tx) context.Context {
	return context.WithValue(ctx, txCtxKey{}, tx)
}

// TxFromContext возвращает транзакцию, если текущий вызов идёт внутри WithinTx.
func TxFromContext(ctx context.Context) (pgx.Tx, bool) {
	tx, ok := ctx.Value(txCtxKey{}).(pgx.Tx)
	return tx, ok
}

type querier interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func querierFrom(ctx context.Context, pool *pgxpool.Pool) querier {
	if tx, ok := TxFromContext(ctx); ok {
		return tx
	}
	return pool
}

// TxManager — postgres-адаптер application.TxManager.
type TxManager struct {
	db *DB
}

// NewTxManager создаёт менеджер транзакций поверх пула.
func NewTxManager(db *DB) *TxManager {
	return &TxManager{db: db}
}

// WithinTx открывает транзакцию, кладёт её в ctx и коммитит при успехе fn.
// Повторный вход в уже открытый tx не начинает вложенную транзакцию.
func (m *TxManager) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	if _, ok := TxFromContext(ctx); ok {
		return fn(ctx)
	}

	tx, err := m.db.Pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := fn(ContextWithTx(ctx, tx)); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}

var _ application.TxManager = (*TxManager)(nil)
