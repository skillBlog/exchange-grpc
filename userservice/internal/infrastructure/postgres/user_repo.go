package postgres

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/exchange-grpc/shared/roles"
	"github.com/exchange-grpc/userservice/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// UserRepository хранит пользователей в PostgreSQL.
type UserRepository struct {
	db *DB
}

// NewUserRepository создаёт postgres user repository.
func NewUserRepository(db *DB) *UserRepository {
	return &UserRepository{db: db}
}

// Save сохраняет пользователя.
func (r *UserRepository) Save(ctx context.Context, user domain.User) error {
	roleStrings := user.RoleStrings()
	_, err := r.db.Pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, roles)
		VALUES ($1, $2, $3, $4)
	`, user.ID, user.Email, user.PasswordHash, roleStrings)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return fmt.Errorf("%w: email already registered", domain.ErrAlreadyExists)
		}
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

// GetByEmail возвращает пользователя по email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	row := r.db.Pool.QueryRow(ctx, `
		SELECT id, email, password_hash, roles
		FROM users
		WHERE email = $1
	`, strings.TrimSpace(strings.ToLower(email)))

	user, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrUnauthorized
	}
	return user, err
}

// GetByID возвращает пользователя по id.
func (r *UserRepository) GetByID(ctx context.Context, id string) (domain.User, error) {
	row := r.db.Pool.QueryRow(ctx, `
		SELECT id, email, password_hash, roles
		FROM users
		WHERE id = $1
	`, id)

	user, err := scanUser(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return domain.User{}, domain.ErrNotFound
	}
	return user, err
}

type userRow interface {
	Scan(dest ...any) error
}

func scanUser(row userRow) (domain.User, error) {
	var (
		id           string
		email        string
		passwordHash string
		roleStrings  []string
	)
	if err := row.Scan(&id, &email, &passwordHash, &roleStrings); err != nil {
		return domain.User{}, err
	}

	parsedRoles := make([]roles.Role, 0, len(roleStrings))
	for _, raw := range roleStrings {
		role, ok := roles.Parse(raw)
		if !ok {
			return domain.User{}, fmt.Errorf("%w: invalid role %q", domain.ErrInvalidArgument, raw)
		}
		parsedRoles = append(parsedRoles, role)
	}

	return domain.NewUser(id, email, passwordHash, parsedRoles)
}
