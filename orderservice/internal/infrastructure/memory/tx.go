package memory

import (
	"context"

	"github.com/exchange-grpc/orderservice/internal/application"
)

// TxManager — in-memory адаптер: транзакций нет, fn вызывается как есть.
// Если Append после Create упадёт, ордер в памяти останется (в отличие от postgres).
type TxManager struct{}

// NewTxManager создаёт passthrough TxManager для тестов.
func NewTxManager() *TxManager {
	return &TxManager{}
}

// WithinTx просто вызывает fn.
func (*TxManager) WithinTx(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

var _ application.TxManager = (*TxManager)(nil)
