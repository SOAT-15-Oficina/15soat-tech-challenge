package repository

import (
	"context"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type WorkOrderStatusHistoryRepository interface {
	Create(ctx context.Context, history *domain.WorkOrderStatusHistory) error
	FindByWorkOrderID(ctx context.Context, workOrderID uuid.UUID) ([]domain.WorkOrderStatusHistory, error)
}

type workOrderStatusHistoryRepository struct {
	db *pgxpool.Pool
}

func NewWorkOrderStatusHistoryRepository(db *pgxpool.Pool) WorkOrderStatusHistoryRepository {
	return &workOrderStatusHistoryRepository{db: db}
}

func (r *workOrderStatusHistoryRepository) Create(ctx context.Context, h *domain.WorkOrderStatusHistory) error {
	query := `
		INSERT INTO work_order_status_history (id, work_order_id, from_status, to_status, changed_by_user_id, note, changed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`
	_, err := r.db.Exec(ctx, query, h.ID, h.WorkOrderID, h.FromStatus, h.ToStatus, h.ChangedByUserID, h.Note, h.ChangedAt)
	return err
}

func (r *workOrderStatusHistoryRepository) FindByWorkOrderID(ctx context.Context, workOrderID uuid.UUID) ([]domain.WorkOrderStatusHistory, error) {
	query := `
		SELECT id, work_order_id, from_status, to_status, changed_by_user_id, note, changed_at
		FROM work_order_status_history
		WHERE work_order_id = $1
		ORDER BY changed_at ASC`

	rows, err := r.db.Query(ctx, query, workOrderID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []domain.WorkOrderStatusHistory
	for rows.Next() {
		var h domain.WorkOrderStatusHistory
		if err := rows.Scan(&h.ID, &h.WorkOrderID, &h.FromStatus, &h.ToStatus, &h.ChangedByUserID, &h.Note, &h.ChangedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}
