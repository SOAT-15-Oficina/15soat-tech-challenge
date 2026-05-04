package repository

import (
	"context"
	"time"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkOrderRepository interface {
	Create(ctx context.Context, workOrder *domain.WorkOrder) (*domain.WorkOrder, error)
	FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrder, error)
	FindByCode(ctx context.Context, code string) (*domain.WorkOrder, error)
	FindAll(ctx context.Context) ([]domain.WorkOrder, error)
	Update(ctx context.Context, workOrder *domain.WorkOrder) (*domain.WorkOrder, error)
}

type workOrderRepository struct {
	db *pgxpool.Pool
}

func NewWorkOrderRepository(db *pgxpool.Pool) WorkOrderRepository {
	return &workOrderRepository{db: db}
}

func (r *workOrderRepository) Create(ctx context.Context, wo *domain.WorkOrder) (*domain.WorkOrder, error) {
	query := `
		INSERT INTO work_orders (
			id, code, title, description, customer_id, vehicle_id, opened_by_user_id, 
			assigned_technician_id, status, total_estimated_price_cents, received_at, 
			quote_sent_at, approved_at, started_at, finished_at, delivered_at, 
			created_at, updated_at
		)
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18
		)
		RETURNING 
			id, code, title, description, customer_id, vehicle_id, opened_by_user_id, 
			assigned_technician_id, status, total_estimated_price_cents, received_at, 
			quote_sent_at, approved_at, started_at, finished_at, delivered_at, 
			created_at, updated_at`

	if wo.ID == uuid.Nil {
		wo.ID = uuid.New()
	}
	now := time.Now()
	if wo.CreatedAt.IsZero() {
		wo.CreatedAt = now
	}
	if wo.UpdatedAt.IsZero() {
		wo.UpdatedAt = now
	}

	var result domain.WorkOrder
	err := r.db.QueryRow(ctx, query,
		wo.ID, wo.Code, wo.Title, wo.Description, wo.CustomerID, wo.VehicleID, wo.OpenedByUserID,
		wo.AssignedTechnicianID, wo.Status, wo.TotalEstimatedPriceCents, wo.ReceivedAt,
		wo.QuoteSentAt, wo.ApprovedAt, wo.StartedAt, wo.FinishedAt, wo.DeliveredAt,
		wo.CreatedAt, wo.UpdatedAt).
		Scan(
			&result.ID, &result.Code, &result.Title, &result.Description, &result.CustomerID, &result.VehicleID, &result.OpenedByUserID,
			&result.AssignedTechnicianID, &result.Status, &result.TotalEstimatedPriceCents, &result.ReceivedAt,
			&result.QuoteSentAt, &result.ApprovedAt, &result.StartedAt, &result.FinishedAt, &result.DeliveredAt,
			&result.CreatedAt, &result.UpdatedAt,
		)

	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *workOrderRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrder, error) {
	query := `
		SELECT 
			id, code, title, description, customer_id, vehicle_id, opened_by_user_id, 
			assigned_technician_id, status, total_estimated_price_cents, received_at, 
			quote_sent_at, approved_at, started_at, finished_at, delivered_at, 
			created_at, updated_at
		FROM work_orders WHERE id = $1`

	var result domain.WorkOrder
	err := r.db.QueryRow(ctx, query, id).
		Scan(
			&result.ID, &result.Code, &result.Title, &result.Description, &result.CustomerID, &result.VehicleID, &result.OpenedByUserID,
			&result.AssignedTechnicianID, &result.Status, &result.TotalEstimatedPriceCents, &result.ReceivedAt,
			&result.QuoteSentAt, &result.ApprovedAt, &result.StartedAt, &result.FinishedAt, &result.DeliveredAt,
			&result.CreatedAt, &result.UpdatedAt,
		)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *workOrderRepository) FindByCode(ctx context.Context, code string) (*domain.WorkOrder, error) {
	query := `
		SELECT
			id, code, title, description, customer_id, vehicle_id, opened_by_user_id,
			assigned_technician_id, status, total_estimated_price_cents, received_at,
			quote_sent_at, approved_at, started_at, finished_at, delivered_at,
			created_at, updated_at
		FROM work_orders WHERE code = $1`

	var result domain.WorkOrder
	err := r.db.QueryRow(ctx, query, code).
		Scan(
			&result.ID, &result.Code, &result.Title, &result.Description, &result.CustomerID, &result.VehicleID, &result.OpenedByUserID,
			&result.AssignedTechnicianID, &result.Status, &result.TotalEstimatedPriceCents, &result.ReceivedAt,
			&result.QuoteSentAt, &result.ApprovedAt, &result.StartedAt, &result.FinishedAt, &result.DeliveredAt,
			&result.CreatedAt, &result.UpdatedAt,
		)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

func (r *workOrderRepository) FindAll(ctx context.Context) ([]domain.WorkOrder, error) {
	query := `
		SELECT 
			id, code, title, description, customer_id, vehicle_id, opened_by_user_id, 
			assigned_technician_id, status, total_estimated_price_cents, received_at, 
			quote_sent_at, approved_at, started_at, finished_at, delivered_at, 
			created_at, updated_at
		FROM work_orders
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var workOrders []domain.WorkOrder
	for rows.Next() {
		var wo domain.WorkOrder
		if err := rows.Scan(
			&wo.ID, &wo.Code, &wo.Title, &wo.Description, &wo.CustomerID, &wo.VehicleID, &wo.OpenedByUserID,
			&wo.AssignedTechnicianID, &wo.Status, &wo.TotalEstimatedPriceCents, &wo.ReceivedAt,
			&wo.QuoteSentAt, &wo.ApprovedAt, &wo.StartedAt, &wo.FinishedAt, &wo.DeliveredAt,
			&wo.CreatedAt, &wo.UpdatedAt,
		); err != nil {
			return nil, err
		}
		workOrders = append(workOrders, wo)
	}
	return workOrders, nil
}

func (r *workOrderRepository) Update(ctx context.Context, wo *domain.WorkOrder) (*domain.WorkOrder, error) {
	query := `
		UPDATE work_orders
		SET 
			code = $1, title = $2, description = $3, customer_id = $4, vehicle_id = $5, 
			opened_by_user_id = $6, assigned_technician_id = $7, status = $8, 
			total_estimated_price_cents = $9, received_at = $10, quote_sent_at = $11, 
			approved_at = $12, started_at = $13, finished_at = $14, delivered_at = $15, 
			updated_at = $16
		WHERE id = $17
		RETURNING 
			id, code, title, description, customer_id, vehicle_id, opened_by_user_id, 
			assigned_technician_id, status, total_estimated_price_cents, received_at, 
			quote_sent_at, approved_at, started_at, finished_at, delivered_at, 
			created_at, updated_at`

	wo.UpdatedAt = time.Now()

	var result domain.WorkOrder
	err := r.db.QueryRow(ctx, query,
		wo.Code, wo.Title, wo.Description, wo.CustomerID, wo.VehicleID, wo.OpenedByUserID,
		wo.AssignedTechnicianID, wo.Status, wo.TotalEstimatedPriceCents, wo.ReceivedAt,
		wo.QuoteSentAt, wo.ApprovedAt, wo.StartedAt, wo.FinishedAt, wo.DeliveredAt,
		wo.UpdatedAt, wo.ID).
		Scan(
			&result.ID, &result.Code, &result.Title, &result.Description, &result.CustomerID, &result.VehicleID, &result.OpenedByUserID,
			&result.AssignedTechnicianID, &result.Status, &result.TotalEstimatedPriceCents, &result.ReceivedAt,
			&result.QuoteSentAt, &result.ApprovedAt, &result.StartedAt, &result.FinishedAt, &result.DeliveredAt,
			&result.CreatedAt, &result.UpdatedAt,
		)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
