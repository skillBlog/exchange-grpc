package postgres

import (
	"context"
	"errors"
	"fmt"

	"github.com/exchange-grpc/spotservice/internal/domain"
	"github.com/jackc/pgx/v5"
)

// MarketRepository хранит рынки в PostgreSQL.
type MarketRepository struct {
	db *DB
}

// NewMarketRepository создаёт postgres market repository.
func NewMarketRepository(db *DB) *MarketRepository {
	return &MarketRepository{db: db}
}

// GetByID возвращает рынок по идентификатору, если он доступен ролям пользователя.
func (r *MarketRepository) GetByID(ctx context.Context, id string, userRoles []string) (domain.Market, error) {
	if userRoles == nil {
		userRoles = []string{}
	}

	row := r.db.Pool.QueryRow(ctx, `
		SELECT id, name, base_asset, quote_asset, enabled, allowed_roles,
			min_order_size::text, quantity_precision, min_notional::text
		FROM markets
		WHERE id = $1
			AND (
				cardinality(allowed_roles) = 0
				OR allowed_roles && $2::text[]
			)
	`, id, userRoles)

	market, err := scanMarket(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Market{}, fmt.Errorf("%w: market %q", domain.ErrNotFound, id)
	}
	return market, err
}

// ListActivePage возвращает страницу активных рынков с RBAC-фильтром и курсором по id.
func (r *MarketRepository) ListActivePage(ctx context.Context, userRoles []string, limit int, afterID string) ([]domain.Market, error) {
	if limit <= 0 {
		return []domain.Market{}, nil
	}
	if userRoles == nil {
		userRoles = []string{}
	}

	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, name, base_asset, quote_asset, enabled, allowed_roles,
			min_order_size::text, quantity_precision, min_notional::text
		FROM markets
		WHERE enabled = TRUE
			AND ($2 = '' OR id > $2)
			AND (
				cardinality(allowed_roles) = 0
				OR allowed_roles && $1::text[]
			)
		ORDER BY id
		LIMIT $3
	`, userRoles, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("list active markets: %w", err)
	}
	defer rows.Close()

	markets := make([]domain.Market, 0, limit)
	for rows.Next() {
		market, err := scanMarket(rows)
		if err != nil {
			return nil, err
		}
		markets = append(markets, market)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return markets, nil
}

// Ping проверяет доступность PostgreSQL.
func (r *MarketRepository) Ping(ctx context.Context) error {
	return r.db.Ping(ctx)
}

type marketRow interface {
	Scan(dest ...any) error
}

func scanMarket(row marketRow) (domain.Market, error) {
	var (
		id, name, baseAsset, quoteAsset string
		enabled                         bool
		allowedRoles                    []string
		minOrderSize                    string
		quantityPrecision               int32
		minNotional                     string
	)
	if err := row.Scan(
		&id,
		&name,
		&baseAsset,
		&quoteAsset,
		&enabled,
		&allowedRoles,
		&minOrderSize,
		&quantityPrecision,
		&minNotional,
	); err != nil {
		return domain.Market{}, err
	}
	var precision uint32
	if quantityPrecision > 0 {
		precision = uint32(quantityPrecision)
	}
	return domain.NewMarket(id, name, baseAsset, quoteAsset, enabled, allowedRoles, domain.Limits{
		MinOrderSize:      minOrderSize,
		QuantityPrecision: precision,
		MinNotional:       minNotional,
	})
}

var _ domain.MarketRepository = (*MarketRepository)(nil)
