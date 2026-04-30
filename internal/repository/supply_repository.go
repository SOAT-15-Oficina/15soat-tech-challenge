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
		INSERT INTO supplies (service_id, item_id, quantity)
		VALUES ($1, $2, $3)
		RETURNING id, service_id, item_id, quantity`

	var result domain.Supply
	err := r.db.QueryRow(ctx, query, supply.ServiceID, supply.ItemID, supply.Quantity).
		Scan(&result.ID, &result.ServiceID, &result.ItemID, &result.Quantity)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *supplyRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Supply, error) {
	query := `SELECT id, service_id, item_id, quantity FROM supplies WHERE id = $1`

	var result domain.Supply
	err := r.db.QueryRow(ctx, query, id).
		Scan(&result.ID, &result.ServiceID, &result.ItemID, &result.Quantity)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *supplyRepository) FindAll(ctx context.Context) ([]domain.Supply, error) {
	query := `SELECT id, service_id, item_id, quantity FROM supplies`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var supplies []domain.Supply
	for rows.Next() {
		var supply domain.Supply
		if err := rows.Scan(&supply.ID, &supply.ServiceID, &supply.ItemID, &supply.Quantity); err != nil {
			return nil, err
		}
		supplies = append(supplies, supply)
	}
	return supplies, nil
}

func (r *supplyRepository) Update(ctx context.Context, supply *domain.Supply) (*domain.Supply, error) {
	query := `
		UPDATE supplies
		SET service_id = $1, item_id = $2, quantity = $3
		WHERE id = $4
		RETURNING id, service_id, item_id, quantity`

	var result domain.Supply
	err := r.db.QueryRow(ctx, query, supply.ServiceID, supply.ItemID, supply.Quantity, supply.ID).
		Scan(&result.ID, &result.ServiceID, &result.ItemID, &result.Quantity)
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
