package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/exchange-grpc/orderservice/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// OrderRepository хранит ордера в PostgreSQL.
type OrderRepository struct {
	db *DB
}

// NewOrderRepository создаёт postgres order repository.
func NewOrderRepository(db *DB) *OrderRepository {
	return &OrderRepository{db: db}
}

// Create сохраняет новый ордер.
func (r *OrderRepository) Create(ctx context.Context, order domain.Order) error {
	var priceAmount, priceCurrency *string
	if !order.Price.IsZero() {
		amount := order.Price.Amount
		currency := order.Price.Currency
		priceAmount = &amount
		priceCurrency = &currency
	}

	_, err := querierFrom(ctx, r.db.Pool).Exec(ctx, `
		INSERT INTO orders (
			id, user_id, market_id, side, price_amount, price_currency, quantity, status, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5::numeric, $6, $7::numeric, $8, $9, $10)
	`, order.ID, order.UserID, order.MarketID, string(order.Side),
		priceAmount, priceCurrency, order.Quantity.Value, string(order.Status),
		order.CreatedAt, order.UpdatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%w: order %q", domain.ErrAlreadyExists, order.ID)
		}
		return fmt.Errorf("insert order: %w", err)
	}
	return nil
}

// GetByID возвращает ордер по id.
func (r *OrderRepository) GetByID(ctx context.Context, id string) (domain.Order, error) {
	row := querierFrom(ctx, r.db.Pool).QueryRow(ctx, `
		SELECT id, user_id, market_id, side,
			price_amount::text, price_currency, quantity::text,
			status, created_at, updated_at
		FROM orders
		WHERE id = $1
	`, id)

	order, err := scanOrder(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, fmt.Errorf("%w: order %q", domain.ErrNotFound, id)
	}
	return order, err
}

// GetByIDAndUserID возвращает ордер при совпадении владельца (один запрос к БД).
func (r *OrderRepository) GetByIDAndUserID(ctx context.Context, orderID, userID string) (domain.Order, error) {
	row := querierFrom(ctx, r.db.Pool).QueryRow(ctx, `
		SELECT id, user_id, market_id, side,
			price_amount::text, price_currency, quantity::text,
			status, created_at, updated_at
		FROM orders
		WHERE id = $1 AND user_id = $2
	`, orderID, userID)

	order, err := scanOrder(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.Order{}, fmt.Errorf("%w: order %q", domain.ErrNotFound, orderID)
	}
	return order, err
}

// ListByUserID возвращает страницу ордеров пользователя по курсору id.
func (r *OrderRepository) ListByUserID(ctx context.Context, userID string, limit int, afterID string, filter domain.ListOrdersFilter) ([]domain.Order, error) {
	if limit <= 0 {
		return []domain.Order{}, nil
	}

	rows, err := querierFrom(ctx, r.db.Pool).Query(ctx, `
		SELECT id, user_id, market_id, side,
			price_amount::text, price_currency, quantity::text,
			status, created_at, updated_at
		FROM orders
		WHERE user_id = $1
			AND ($2 = '' OR id > $2::uuid)
			AND ($4 = '' OR market_id = $4)
			AND ($5 = '' OR status = $5)
		ORDER BY id
		LIMIT $3
	`, userID, afterID, limit, filter.MarketID, string(filter.Status))
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()

	orders := make([]domain.Order, 0, limit)
	for rows.Next() {
		order, err := scanOrder(rows)
		if err != nil {
			return nil, err
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}

// UpdateStatus обновляет статус ордера, если он всё ещё равен expected.
func (r *OrderRepository) UpdateStatus(ctx context.Context, id string, expected, next domain.OrderStatus, updatedAt time.Time) error {
	q := querierFrom(ctx, r.db.Pool)
	tag, err := q.Exec(ctx, `
		UPDATE orders
		SET status = $2, updated_at = $3
		WHERE id = $1 AND status = $4
	`, id, string(next), updatedAt, string(expected))
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}

	var current string
	err = q.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1`, id).Scan(&current)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: order %q", domain.ErrNotFound, id)
	}
	if err != nil {
		return fmt.Errorf("update order status: %w", err)
	}
	return fmt.Errorf("%w: order %q status is %s", domain.ErrConflict, id, current)
}

type orderRow interface {
	Scan(dest ...any) error
}

func scanOrder(row orderRow) (domain.Order, error) {
	var (
		order         domain.Order
		side          string
		status        string
		priceAmount   *string
		priceCurrency *string
		quantity      string
	)
	if err := row.Scan(
		&order.ID,
		&order.UserID,
		&order.MarketID,
		&side,
		&priceAmount,
		&priceCurrency,
		&quantity,
		&status,
		&order.CreatedAt,
		&order.UpdatedAt,
	); err != nil {
		return domain.Order{}, err
	}

	order.Side = domain.OrderSide(side)
	order.Status = domain.OrderStatus(status)
	order.Quantity = domain.Decimal{Value: quantity}
	if priceAmount != nil && *priceAmount != "" {
		currency := "USD"
		if priceCurrency != nil {
			currency = *priceCurrency
		}
		money, err := domain.NewMoney(*priceAmount, currency)
		if err != nil {
			return domain.Order{}, err
		}
		order.Price = money
	}

	return order, nil
}

var _ domain.OrderRepository = (*OrderRepository)(nil)
