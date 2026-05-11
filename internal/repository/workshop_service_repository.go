package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkshopServiceRepository interface {
	Create(ctx context.Context, ws *domain.WorkshopService) (*domain.WorkshopService, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkshopService, error)
	List(ctx context.Context, filters domain.WorkshopServiceListFilters) ([]domain.WorkshopService, int, error)
	Update(ctx context.Context, ws *domain.WorkshopService) (*domain.WorkshopService, error)
	Delete(ctx context.Context, id uuid.UUID) error
	Deactivate(ctx context.Context, id uuid.UUID) (*domain.WorkshopService, error)
	ExistsByTitle(ctx context.Context, title string, excludeID *uuid.UUID) (bool, error)
	HasWorkOrderLinks(ctx context.Context, id uuid.UUID) (bool, error)
	SubtractSuppliesFromStock(ctx context.Context, serviceID uuid.UUID) error
}

type workshopServiceRepository struct {
	db *pgxpool.Pool
}

func NewWorkshopServiceRepository(db *pgxpool.Pool) WorkshopServiceRepository {
	return &workshopServiceRepository{db: db}
}

func (r *workshopServiceRepository) Create(ctx context.Context, ws *domain.WorkshopService) (*domain.WorkshopService, error) {
	query := `
		INSERT INTO services (id, title, description, price_cents, estimated_time_minutes, active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, title, description, price_cents, estimated_time_minutes, active, created_at, updated_at`

	now := time.Now().UTC()
	if ws.ID == uuid.Nil {
		ws.ID = uuid.New()
	}
	ws.CreatedAt = now
	ws.UpdatedAt = now

	var result domain.WorkshopService
	err := r.db.QueryRow(ctx, query,
		ws.ID, ws.Title, ws.Description, ws.PriceCents,
		ws.EstimatedTimeMinutes, ws.Active, ws.CreatedAt, ws.UpdatedAt,
	).Scan(
		&result.ID, &result.Title, &result.Description, &result.PriceCents,
		&result.EstimatedTimeMinutes, &result.Active, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *workshopServiceRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkshopService, error) {
	query := `
		SELECT id, title, description, price_cents, estimated_time_minutes, active, created_at, updated_at
		FROM services WHERE id = $1`

	var result domain.WorkshopService
	err := r.db.QueryRow(ctx, query, id).Scan(
		&result.ID, &result.Title, &result.Description, &result.PriceCents,
		&result.EstimatedTimeMinutes, &result.Active, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *workshopServiceRepository) List(ctx context.Context, filters domain.WorkshopServiceListFilters) ([]domain.WorkshopService, int, error) {
	where := []string{"1 = 1"}
	args := []any{}

	if filters.Active != nil {
		args = append(args, *filters.Active)
		where = append(where, fmt.Sprintf("active = $%d", len(args)))
	}

	if filters.Title != "" {
		args = append(args, "%"+strings.TrimSpace(filters.Title)+"%")
		where = append(where, fmt.Sprintf("title ILIKE $%d", len(args)))
	}

	whereClause := strings.Join(where, " AND ")

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM services WHERE %s", whereClause)
	var total int
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, filters.Limit, (filters.Page-1)*filters.Limit)
	listQuery := fmt.Sprintf(`
		SELECT id, title, description, price_cents, estimated_time_minutes, active, created_at, updated_at
		FROM services
		WHERE %s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d`, whereClause, len(args)-1, len(args))

	rows, err := r.db.Query(ctx, listQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var services []domain.WorkshopService
	for rows.Next() {
		var item domain.WorkshopService
		if err := rows.Scan(
			&item.ID, &item.Title, &item.Description, &item.PriceCents,
			&item.EstimatedTimeMinutes, &item.Active, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		services = append(services, item)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	return services, total, nil
}

func (r *workshopServiceRepository) Update(ctx context.Context, ws *domain.WorkshopService) (*domain.WorkshopService, error) {
	query := `
		UPDATE services
		SET title = $1, description = $2, price_cents = $3, estimated_time_minutes = $4, active = $5, updated_at = $6
		WHERE id = $7
		RETURNING id, title, description, price_cents, estimated_time_minutes, active, created_at, updated_at`

	ws.UpdatedAt = time.Now().UTC()

	var result domain.WorkshopService
	err := r.db.QueryRow(ctx, query,
		ws.Title, ws.Description, ws.PriceCents, ws.EstimatedTimeMinutes,
		ws.Active, ws.UpdatedAt, ws.ID,
	).Scan(
		&result.ID, &result.Title, &result.Description, &result.PriceCents,
		&result.EstimatedTimeMinutes, &result.Active, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *workshopServiceRepository) Delete(ctx context.Context, id uuid.UUID) error {
	commandTag, err := r.db.Exec(ctx, `DELETE FROM services WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}

	return nil
}

func (r *workshopServiceRepository) Deactivate(ctx context.Context, id uuid.UUID) (*domain.WorkshopService, error) {
	query := `
		UPDATE services SET active = false, updated_at = $1 WHERE id = $2
		RETURNING id, title, description, price_cents, estimated_time_minutes, active, created_at, updated_at`

	var result domain.WorkshopService
	err := r.db.QueryRow(ctx, query, time.Now().UTC(), id).Scan(
		&result.ID, &result.Title, &result.Description, &result.PriceCents,
		&result.EstimatedTimeMinutes, &result.Active, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return &result, nil
}

func (r *workshopServiceRepository) ExistsByTitle(ctx context.Context, title string, excludeID *uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM services WHERE LOWER(title) = LOWER($1))`
	args := []any{title}

	if excludeID != nil {
		query = `SELECT EXISTS(SELECT 1 FROM services WHERE LOWER(title) = LOWER($1) AND id <> $2)`
		args = append(args, *excludeID)
	}

	var exists bool
	if err := r.db.QueryRow(ctx, query, args...).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func (r *workshopServiceRepository) HasWorkOrderLinks(ctx context.Context, id uuid.UUID) (bool, error) {
	query := `SELECT EXISTS(SELECT 1 FROM work_order_services WHERE service_id = $1)`

	var exists bool
	if err := r.db.QueryRow(ctx, query, id).Scan(&exists); err != nil {
		return false, err
	}

	return exists, nil
}

func (r *workshopServiceRepository) SubtractSuppliesFromStock(ctx context.Context, serviceID uuid.UUID) error {
	query := `
		UPDATE supplies s
		SET stock_quantity = s.stock_quantity - woss.supply_quantity
		FROM work_order_service_supplies woss
		JOIN work_order_services wos ON wos.id = woss.work_order_service_id
		WHERE woss.supply_id = s.id
		  AND wos.service_id = $1`

	_, err := r.db.Exec(ctx, query, serviceID)
	return err
}
