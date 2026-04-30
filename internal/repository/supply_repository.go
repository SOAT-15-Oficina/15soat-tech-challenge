package repository

import (
	"context"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SupplyRepository interface {
	Create(ctx context.Context, supply *domain.Supply) (*domain.Supply, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Supply, error)
	FindAll(ctx context.Context) ([]domain.Supply, error)
	Update(ctx context.Context, supply *domain.Supply) (*domain.Supply, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type supplyRepository struct {
	db *pgxpool.Pool
}

func NewSupplyRepository(db *pgxpool.Pool) SupplyRepository {
	return &supplyRepository{db: db}
}

func (r *supplyRepository) Create(ctx context.Context, supply *domain.Supply) (*domain.Supply, error) {
	query := `
		INSERT INTO supplies (id, title, type, price_cents, stock_quantity, minimum_stock, active, created_at, updated_at)
		VALUES (COALESCE($1, gen_random_uuid()), $2, $3, $4, $5, $6, $7, NOW(), NOW())
		RETURNING id, title, type, price_cents, stock_quantity, minimum_stock, active, created_at, updated_at`

	var idArg any
	if supply.ID != uuid.Nil {
		idArg = supply.ID
	}

	var result domain.Supply
	err := r.db.QueryRow(ctx, query,
		idArg,
		supply.Title,
		supply.Type,
		supply.PriceCents,
		supply.StockQuantity,
		supply.MinimumStock,
		supply.Active,
	).Scan(
		&result.ID,
		&result.Title,
		&result.Type,
		&result.PriceCents,
		&result.StockQuantity,
		&result.MinimumStock,
		&result.Active,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *supplyRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Supply, error) {
	query := `SELECT id, title, type, price_cents, stock_quantity, minimum_stock, active, created_at, updated_at
		FROM supplies WHERE id = $1`

	var result domain.Supply
	err := r.db.QueryRow(ctx, query, id).Scan(
		&result.ID,
		&result.Title,
		&result.Type,
		&result.PriceCents,
		&result.StockQuantity,
		&result.MinimumStock,
		&result.Active,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *supplyRepository) FindAll(ctx context.Context) ([]domain.Supply, error) {
	query := `SELECT id, title, type, price_cents, stock_quantity, minimum_stock, active, created_at, updated_at
		FROM supplies`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var supplies []domain.Supply
	for rows.Next() {
		var supply domain.Supply
		if err := rows.Scan(
			&supply.ID,
			&supply.Title,
			&supply.Type,
			&supply.PriceCents,
			&supply.StockQuantity,
			&supply.MinimumStock,
			&supply.Active,
			&supply.CreatedAt,
			&supply.UpdatedAt,
		); err != nil {
			return nil, err
		}
		supplies = append(supplies, supply)
	}
	return supplies, nil
}

func (r *supplyRepository) Update(ctx context.Context, supply *domain.Supply) (*domain.Supply, error) {
	query := `
		UPDATE supplies
		SET title = $1, type = $2, price_cents = $3, stock_quantity = $4, minimum_stock = $5, active = $6, updated_at = NOW()
		WHERE id = $7
		RETURNING id, title, type, price_cents, stock_quantity, minimum_stock, active, created_at, updated_at`

	var result domain.Supply
	err := r.db.QueryRow(ctx, query,
		supply.Title,
		supply.Type,
		supply.PriceCents,
		supply.StockQuantity,
		supply.MinimumStock,
		supply.Active,
		supply.ID,
	).Scan(
		&result.ID,
		&result.Title,
		&result.Type,
		&result.PriceCents,
		&result.StockQuantity,
		&result.MinimumStock,
		&result.Active,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *supplyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM supplies WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
