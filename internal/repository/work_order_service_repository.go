package repository

import (
	"context"
	"time"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkOrderServiceRepository interface {
	FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrderService, error)
	FindByWorkOrderID(ctx context.Context, workOrderID uuid.UUID) ([]domain.WorkOrderService, error)
	UpdateApprovalStatus(ctx context.Context, id uuid.UUID, status domain.WorkOrderServiceApprovalStatus) error
	UpdateApprovalStatusByWorkOrderID(ctx context.Context, workOrderID uuid.UUID, status domain.WorkOrderServiceApprovalStatus) error
	CalculateTotalForWorkOrder(ctx context.Context, workOrderID uuid.UUID) (int, error)
	CalculateApprovedTotalForWorkOrder(ctx context.Context, workOrderID uuid.UUID) (int, error)
}

type workOrderServiceRepository struct {
	db *pgxpool.Pool
}

func NewWorkOrderServiceRepository(db *pgxpool.Pool) WorkOrderServiceRepository {
	return &workOrderServiceRepository{db: db}
}

func (r *workOrderServiceRepository) FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrderService, error) {
	query := `
		SELECT id, work_order_id, service_id, service_title_snapshot, service_description_snapshot,
			service_price_cents_snapshot, service_estimated_time_minutes_snapshot,
			approval_status, status, started_at, finished_at, created_at, updated_at
		FROM work_order_services WHERE id = $1`

	var wos domain.WorkOrderService
	err := r.db.QueryRow(ctx, query, id).Scan(
		&wos.ID, &wos.WorkOrderID, &wos.ServiceID, &wos.ServiceTitleSnapshot, &wos.ServiceDescriptionSnapshot,
		&wos.ServicePriceCentsSnapshot, &wos.ServiceEstimatedTimeMinutesSnapshot,
		&wos.ApprovalStatus, &wos.Status, &wos.StartedAt, &wos.FinishedAt, &wos.CreatedAt, &wos.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &wos, nil
}

func (r *workOrderServiceRepository) FindByWorkOrderID(ctx context.Context, workOrderID uuid.UUID) ([]domain.WorkOrderService, error) {
	query := `
		SELECT id, work_order_id, service_id, service_title_snapshot, service_description_snapshot,
			service_price_cents_snapshot, service_estimated_time_minutes_snapshot,
			approval_status, status, started_at, finished_at, created_at, updated_at
		FROM work_order_services WHERE work_order_id = $1`

	rows, err := r.db.Query(ctx, query, workOrderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var services []domain.WorkOrderService
	for rows.Next() {
		var wos domain.WorkOrderService
		if err := rows.Scan(
			&wos.ID, &wos.WorkOrderID, &wos.ServiceID, &wos.ServiceTitleSnapshot, &wos.ServiceDescriptionSnapshot,
			&wos.ServicePriceCentsSnapshot, &wos.ServiceEstimatedTimeMinutesSnapshot,
			&wos.ApprovalStatus, &wos.Status, &wos.StartedAt, &wos.FinishedAt, &wos.CreatedAt, &wos.UpdatedAt,
		); err != nil {
			return nil, err
		}
		services = append(services, wos)
	}
	return services, nil
}

func (r *workOrderServiceRepository) UpdateApprovalStatus(ctx context.Context, id uuid.UUID, status domain.WorkOrderServiceApprovalStatus) error {
	query := `UPDATE work_order_services SET approval_status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.Exec(ctx, query, status, time.Now(), id)
	return err
}

func (r *workOrderServiceRepository) UpdateApprovalStatusByWorkOrderID(ctx context.Context, workOrderID uuid.UUID, status domain.WorkOrderServiceApprovalStatus) error {
	query := `UPDATE work_order_services SET approval_status = $1, updated_at = $2 WHERE work_order_id = $3 AND approval_status = $4`
	_, err := r.db.Exec(ctx, query, status, time.Now(), workOrderID, domain.WorkOrderServiceApprovalPending)
	return err
}

func (r *workOrderServiceRepository) CalculateTotalForWorkOrder(ctx context.Context, workOrderID uuid.UUID) (int, error) {
	query := `
		SELECT COALESCE(SUM(wos.service_price_cents_snapshot), 0) +
			COALESCE((
				SELECT SUM(woss.supply_price_cents_snapshot * woss.supply_quantity)
				FROM work_order_service_supplies woss
				JOIN work_order_services wos2 ON wos2.id = woss.work_order_service_id
				WHERE wos2.work_order_id = $1
			), 0)
		FROM work_order_services wos
		WHERE wos.work_order_id = $1`

	var total int
	err := r.db.QueryRow(ctx, query, workOrderID).Scan(&total)
	return total, err
}

func (r *workOrderServiceRepository) CalculateApprovedTotalForWorkOrder(ctx context.Context, workOrderID uuid.UUID) (int, error) {
	query := `
		SELECT COALESCE(SUM(wos.service_price_cents_snapshot), 0) +
			COALESCE((
				SELECT SUM(woss.supply_price_cents_snapshot * woss.supply_quantity)
				FROM work_order_service_supplies woss
				JOIN work_order_services wos2 ON wos2.id = woss.work_order_service_id
				WHERE wos2.work_order_id = $1 AND wos2.approval_status = 'APROVADO'
			), 0)
		FROM work_order_services wos
		WHERE wos.work_order_id = $1 AND wos.approval_status = 'APROVADO'`

	var total int
	err := r.db.QueryRow(ctx, query, workOrderID).Scan(&total)
	return total, err
}
