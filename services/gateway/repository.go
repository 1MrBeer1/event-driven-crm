package gateway

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mr-beer/event-driven-crm/internal/models"
)

var ErrUserNotFound = errors.New("user not found")

type UserRepository interface {
	Create(ctx context.Context, user models.User) (models.User, error)
	FindByEmail(ctx context.Context, email string) (models.User, error)
}

type PostgresUserRepository struct {
	db *pgxpool.Pool
}

func NewPostgresUserRepository(db *pgxpool.Pool) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

func (r *PostgresUserRepository) Create(ctx context.Context, user models.User) (models.User, error) {
	const query = `
		INSERT INTO users (id, email, password_hash, role)
		VALUES ($1, $2, $3, $4)
		RETURNING id, email, password_hash, role, created_at`

	err := r.db.QueryRow(ctx, query, user.ID, user.Email, user.PasswordHash, user.Role).
		Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt)
	return user, err
}

func (r *PostgresUserRepository) FindByEmail(ctx context.Context, email string) (models.User, error) {
	const query = `
		SELECT id, email, password_hash, role, created_at
		FROM users
		WHERE email = $1`

	var user models.User
	err := r.db.QueryRow(ctx, query, email).
		Scan(&user.ID, &user.Email, &user.PasswordHash, &user.Role, &user.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.User{}, ErrUserNotFound
	}
	return user, err
}
