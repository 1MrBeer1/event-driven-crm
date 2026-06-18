package leadservice

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/mr-beer/event-driven-crm/internal/models"
)

var ErrLeadNotFound = errors.New("lead not found")

type Repository interface {
	Create(ctx context.Context, lead models.Lead) (models.Lead, error)
	Get(ctx context.Context, id string) (models.Lead, error)
	List(ctx context.Context, userID string, limit, offset int) ([]models.Lead, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) Create(ctx context.Context, lead models.Lead) (models.Lead, error) {
	const query = `
		INSERT INTO leads (id, user_id, name, email, company, source, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, user_id, name, email, company, source, status, created_at, updated_at`

	err := r.db.QueryRow(ctx, query,
		lead.ID,
		lead.UserID,
		lead.Name,
		lead.Email,
		lead.Company,
		lead.Source,
		lead.Status,
	).Scan(&lead.ID, &lead.UserID, &lead.Name, &lead.Email, &lead.Company, &lead.Source, &lead.Status, &lead.CreatedAt, &lead.UpdatedAt)
	return lead, err
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (models.Lead, error) {
	const query = `
		SELECT id, user_id, name, email, company, source, status, created_at, updated_at
		FROM leads
		WHERE id = $1`

	var lead models.Lead
	err := r.db.QueryRow(ctx, query, id).
		Scan(&lead.ID, &lead.UserID, &lead.Name, &lead.Email, &lead.Company, &lead.Source, &lead.Status, &lead.CreatedAt, &lead.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Lead{}, ErrLeadNotFound
	}
	return lead, err
}

func (r *PostgresRepository) List(ctx context.Context, userID string, limit, offset int) ([]models.Lead, error) {
	const query = `
		SELECT id, user_id, name, email, company, source, status, created_at, updated_at
		FROM leads
		WHERE ($1 = '' OR user_id = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	leads := make([]models.Lead, 0)
	for rows.Next() {
		var lead models.Lead
		if err := rows.Scan(&lead.ID, &lead.UserID, &lead.Name, &lead.Email, &lead.Company, &lead.Source, &lead.Status, &lead.CreatedAt, &lead.UpdatedAt); err != nil {
			return nil, err
		}
		leads = append(leads, lead)
	}
	return leads, rows.Err()
}
