package notificationservice

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mr-beer/event-driven-crm/internal/models"
)

var ErrNotificationNotFound = errors.New("notification not found")

type Repository interface {
	Create(ctx context.Context, notification models.Notification) (models.Notification, error)
	Get(ctx context.Context, id string) (models.Notification, error)
	List(ctx context.Context, userID string, limit, offset int) ([]models.Notification, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, notification models.Notification) (models.Notification, error) {
	const query = `
		INSERT INTO notifications (id, user_id, customer_id, type, message, is_read)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, user_id, customer_id, type, message, is_read, created_at`

	err := r.db.QueryRow(ctx, query,
		notification.ID,
		notification.UserID,
		notification.CustomerID,
		notification.Type,
		notification.Message,
		notification.Read,
	).Scan(&notification.ID, &notification.UserID, &notification.CustomerID, &notification.Type, &notification.Message, &notification.Read, &notification.CreatedAt)
	return notification, err
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (models.Notification, error) {
	const query = `
		SELECT id, user_id, customer_id, type, message, is_read, created_at
		FROM notifications
		WHERE id = $1`

	var notification models.Notification
	err := r.db.QueryRow(ctx, query, id).
		Scan(&notification.ID, &notification.UserID, &notification.CustomerID, &notification.Type, &notification.Message, &notification.Read, &notification.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Notification{}, ErrNotificationNotFound
	}
	return notification, err
}

func (r *PostgresRepository) List(ctx context.Context, userID string, limit, offset int) ([]models.Notification, error) {
	const query = `
		SELECT id, user_id, customer_id, type, message, is_read, created_at
		FROM notifications
		WHERE ($1 = '' OR user_id = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	notifications := make([]models.Notification, 0)
	for rows.Next() {
		var notification models.Notification
		if err := rows.Scan(&notification.ID, &notification.UserID, &notification.CustomerID, &notification.Type, &notification.Message, &notification.Read, &notification.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, notification)
	}
	return notifications, rows.Err()
}
