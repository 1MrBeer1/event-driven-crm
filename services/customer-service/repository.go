package customerservice

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/1MrBeer1/event-driven-crm/internal/models"
)

var ErrCustomerNotFound = errors.New("customer not found")

type Repository interface {
	CreateFromLead(ctx context.Context, customer models.Customer) (models.Customer, bool, error)
	Get(ctx context.Context, id string) (models.Customer, error)
	List(ctx context.Context, userID string, limit, offset int) ([]models.Customer, error)
}

type PostgresRepository struct {
	db *pgxpool.Pool
}

func NewPostgresRepository(db *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{db: db}
}

func (r *PostgresRepository) CreateFromLead(ctx context.Context, customer models.Customer) (models.Customer, bool, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return models.Customer{}, false, err
	}
	defer tx.Rollback(ctx)

	const insertQuery = `
		INSERT INTO customers (id, lead_id, user_id, name, email, company)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (lead_id) DO NOTHING
		RETURNING id, lead_id, user_id, name, email, company, created_at, updated_at`

	err = tx.QueryRow(ctx, insertQuery,
		customer.ID,
		customer.LeadID,
		customer.UserID,
		customer.Name,
		customer.Email,
		customer.Company,
	).Scan(&customer.ID, &customer.LeadID, &customer.UserID, &customer.Name, &customer.Email, &customer.Company, &customer.CreatedAt, &customer.UpdatedAt)

	created := true
	if errors.Is(err, pgx.ErrNoRows) {
		created = false
		customer, err = r.getByLeadID(ctx, tx, customer.LeadID)
	}
	if err != nil {
		return models.Customer{}, false, err
	}

	if created {
		if _, err := tx.Exec(ctx, `UPDATE leads SET status = 'converted', updated_at = now() WHERE id = $1`, customer.LeadID); err != nil {
			return models.Customer{}, false, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return models.Customer{}, false, err
	}
	return customer, created, nil
}

func (r *PostgresRepository) getByLeadID(ctx context.Context, q pgx.Tx, leadID string) (models.Customer, error) {
	const query = `
		SELECT id, lead_id, user_id, name, email, company, created_at, updated_at
		FROM customers
		WHERE lead_id = $1`

	var customer models.Customer
	err := q.QueryRow(ctx, query, leadID).
		Scan(&customer.ID, &customer.LeadID, &customer.UserID, &customer.Name, &customer.Email, &customer.Company, &customer.CreatedAt, &customer.UpdatedAt)
	return customer, err
}

func (r *PostgresRepository) Get(ctx context.Context, id string) (models.Customer, error) {
	const query = `
		SELECT id, lead_id, user_id, name, email, company, created_at, updated_at
		FROM customers
		WHERE id = $1`

	var customer models.Customer
	err := r.db.QueryRow(ctx, query, id).
		Scan(&customer.ID, &customer.LeadID, &customer.UserID, &customer.Name, &customer.Email, &customer.Company, &customer.CreatedAt, &customer.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return models.Customer{}, ErrCustomerNotFound
	}
	return customer, err
}

func (r *PostgresRepository) List(ctx context.Context, userID string, limit, offset int) ([]models.Customer, error) {
	const query = `
		SELECT id, lead_id, user_id, name, email, company, created_at, updated_at
		FROM customers
		WHERE ($1 = '' OR user_id = $1)
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	customers := make([]models.Customer, 0)
	for rows.Next() {
		var customer models.Customer
		if err := rows.Scan(&customer.ID, &customer.LeadID, &customer.UserID, &customer.Name, &customer.Email, &customer.Company, &customer.CreatedAt, &customer.UpdatedAt); err != nil {
			return nil, err
		}
		customers = append(customers, customer)
	}
	return customers, rows.Err()
}
