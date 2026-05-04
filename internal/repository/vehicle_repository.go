package repository

import (
	"context"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type VehicleRepository interface {
	Create(ctx context.Context, vehicle *domain.Vehicle) (*domain.Vehicle, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Vehicle, error)
	FindAll(ctx context.Context) ([]domain.Vehicle, error)
	Update(ctx context.Context, vehicle *domain.Vehicle) (*domain.Vehicle, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type vehicleRepository struct {
	db *pgxpool.Pool
}

func NewVehicleRepository(db *pgxpool.Pool) VehicleRepository {
	return &vehicleRepository{db: db}
}

func (r *vehicleRepository) Create(ctx context.Context, vehicle *domain.Vehicle) (*domain.Vehicle, error) {
	if vehicle.ID == uuid.Nil {
		vehicle.ID = uuid.New()
	}

	query := `
		INSERT INTO vehicles (id, license_plate, customer_id, model, year, brand, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
		RETURNING id, license_plate, customer_id, model, year, brand, created_at, updated_at`
	
	var result domain.Vehicle
	err := r.db.QueryRow(ctx, query, vehicle.ID, vehicle.LicensePlate, vehicle.CustomerID, vehicle.Model, vehicle.Year, vehicle.Brand).
	Scan(&result.ID, &result.LicensePlate, &result.CustomerID, &result.Model, &result.Year, &result.Brand, &result.CreatedAt, &result.UpdatedAt)

	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *vehicleRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.Vehicle, error) {
	query := `SELECT id, license_plate, customer_id, model, year, brand FROM vehicles WHERE id = $1`

	var result domain.Vehicle
	err := r.db.QueryRow(ctx, query, id).
		Scan(&result.ID, &result.LicensePlate, &result.CustomerID, &result.Model, &result.Year, &result.Brand)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *vehicleRepository) FindAll(ctx context.Context) ([]domain.Vehicle, error) {
	query := `SELECT id, license_plate, customer_id, model, year, brand FROM vehicles`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var vehicles []domain.Vehicle
	for rows.Next() {
		var vehicle domain.Vehicle
		if err := rows.Scan(&vehicle.ID, &vehicle.LicensePlate, &vehicle.CustomerID, &vehicle.Model, &vehicle.Year, &vehicle.Brand); err != nil {
			return nil, err
		}
		vehicles = append(vehicles, vehicle)
	}
	return vehicles, nil
}

func (r *vehicleRepository) Update(ctx context.Context, vehicle *domain.Vehicle) (*domain.Vehicle, error) {
	query := `
		UPDATE vehicles
		SET license_plate = $1, customer_id = $2, model = $3, year = $4, brand = $5, updated_at = NOW()
		WHERE id = $6
		RETURNING id, license_plate, customer_id, model, year, brand, created_at, updated_at`

	var result domain.Vehicle
	err := r.db.QueryRow(ctx, query, vehicle.LicensePlate, vehicle.CustomerID, vehicle.Model, vehicle.Year, vehicle.Brand, vehicle.ID).
		Scan(&result.ID, &result.LicensePlate, &result.CustomerID, &result.Model, &result.Year, &result.Brand, &result.CreatedAt, &result.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *vehicleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM vehicles WHERE id = $1`
	_, err := r.db.Exec(ctx, query, id)
	return err
}
