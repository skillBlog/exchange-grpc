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

// GetByID возвращает рынок по идентификатору.
func (r *MarketRepository) GetByID(ctx context.Context, id string) (domain.Market, error) {
	row := r.db.Pool.QueryRow(ctx, `
		SELECT id, name, base_asset, quote_asset, enabled, allowed_roles
		FROM markets
		WHERE id = $1
	`, id)

	market, err := scanMarket(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Market{}, fmt.Errorf("%w: market %q", domain.ErrNotFound, id)
	}
	return market, err
}

// ListActive возвращает включённые рынки.
func (r *MarketRepository) ListActive(ctx context.Context) ([]domain.Market, error) {
	rows, err := r.db.Pool.Query(ctx, `
		SELECT id, name, base_asset, quote_asset, enabled, allowed_roles
		FROM markets
		WHERE enabled = TRUE
		ORDER BY id
	`)
	if err != nil {
		return nil, fmt.Errorf("list active markets: %w", err)
	}
	defer rows.Close()

	markets := make([]domain.Market, 0)
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

type marketRow interface {
	Scan(dest ...any) error
}

func scanMarket(row marketRow) (domain.Market, error) {
	var market domain.Market
	if err := row.Scan(
		&market.ID,
		&market.Name,
		&market.BaseAsset,
		&market.QuoteAsset,
		&market.Enabled,
		&market.AllowedRoles,
	); err != nil {
		return domain.Market{}, err
	}
	return market, nil
}

var _ domain.MarketRepository = (*MarketRepository)(nil)
