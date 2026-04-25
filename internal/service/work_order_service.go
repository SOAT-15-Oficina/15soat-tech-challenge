package service

import (
	"context"
	"errors"
	"time"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/google/uuid"
)

type WorkOrderService interface {
	Create(ctx context.Context, workOrder *domain.WorkOrder) (*domain.WorkOrder, error)
	GetByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrder, error)
	GetAll(ctx context.Context) ([]domain.WorkOrder, error)
	Update(ctx context.Context, workOrder *domain.WorkOrder) (*domain.WorkOrder, error)
}

type workOrderService struct {
	repo repository.WorkOrderRepository
}

func NewWorkOrderService(repo repository.WorkOrderRepository) WorkOrderService {
	return &workOrderService{repo: repo}
}

func (s *workOrderService) Create(ctx context.Context, wo *domain.WorkOrder) (*domain.WorkOrder, error) {
	if wo.ID == uuid.Nil {
		wo.ID = uuid.New()
	}

	if wo.Status == "" {
		wo.Status = domain.WorkOrderStatusReceived
	}

	now := time.Now()
	wo.CreatedAt = now
	wo.UpdatedAt = now

	if wo.ReceivedAt.IsZero() {
		wo.ReceivedAt = now
	}

	if wo.Code == "" {
		return nil, errors.New("code is required")
	}
	if wo.Title == "" {
		return nil, errors.New("title is required")
	}
	if wo.CustomerID == uuid.Nil {
		return nil, errors.New("customer_id is required")
	}
	if wo.VehicleID == uuid.Nil {
		return nil, errors.New("vehicle_id is required")
	}
	if wo.OpenedByUserID == uuid.Nil {
		return nil, errors.New("opened_by_user_id is required")
	}

	return s.repo.Create(ctx, wo)
}

func (s *workOrderService) GetByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrder, error) {
	return s.repo.FindByID(ctx, id)
}

func (s *workOrderService) GetAll(ctx context.Context) ([]domain.WorkOrder, error) {
	return s.repo.FindAll(ctx)
}

func (s *workOrderService) Update(ctx context.Context, wo *domain.WorkOrder) (*domain.WorkOrder, error) {
	existing, err := s.repo.FindByID(ctx, wo.ID)
	if err != nil {
		return nil, err
	}
	wo.CreatedAt = existing.CreatedAt
	wo.UpdatedAt = time.Now()

	if wo.Code != "" { existing.Code = wo.Code }
	if wo.Title != "" { existing.Title = wo.Title }
	if wo.Description != nil { existing.Description = wo.Description }
	if wo.CustomerID != uuid.Nil { existing.CustomerID = wo.CustomerID }
	if wo.VehicleID != uuid.Nil { existing.VehicleID = wo.VehicleID }
	if wo.AssignedTechnicianID != nil { existing.AssignedTechnicianID = wo.AssignedTechnicianID }
	if wo.Status != "" { existing.Status = wo.Status }
	if wo.TotalEstimatedPriceCents != 0 { existing.TotalEstimatedPriceCents = wo.TotalEstimatedPriceCents }
	if !wo.ReceivedAt.IsZero() { existing.ReceivedAt = wo.ReceivedAt }
	if wo.QuoteSentAt != nil { existing.QuoteSentAt = wo.QuoteSentAt }
	if wo.ApprovedAt != nil { existing.ApprovedAt = wo.ApprovedAt }
	if wo.StartedAt != nil { existing.StartedAt = wo.StartedAt }
	if wo.FinishedAt != nil { existing.FinishedAt = wo.FinishedAt }
	if wo.DeliveredAt != nil { existing.DeliveredAt = wo.DeliveredAt }

	return s.repo.Update(ctx, existing)
}
