package service

import (
	"context"
	"errors"
	"fmt"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrWorkOrderInvalidStatusForItems = errors.New("work order status does not allow adding services")
	ErrWorkshopServiceInactive        = errors.New("workshop service is inactive")
	ErrWorkOrderServiceOwnership      = errors.New("work order service does not belong to this work order")
)

type AddWorkOrderServiceInput struct {
	ServiceID            uuid.UUID
	EstimatedTimeMinutes *int
}

type AddWorkOrderSupplyInput struct {
	SupplyID uuid.UUID
	Quantity int
}

type WorkOrderCreationService interface {
	AddServices(ctx context.Context, workOrderID uuid.UUID, items []AddWorkOrderServiceInput) ([]domain.WorkOrderService, error)
	AddSupplies(ctx context.Context, workOrderID, wosID uuid.UUID, items []AddWorkOrderSupplyInput) ([]domain.WorkOrderServiceSupply, error)
	RemoveService(ctx context.Context, workOrderID, wosID uuid.UUID) error
}

type workOrderCreationService struct {
	woRepo     repository.WorkOrderRepository
	wosRepo    repository.WorkOrderServiceRepository
	wsRepo     repository.WorkshopServiceRepository
	supplyRepo repository.SupplyRepository
}

func NewWorkOrderCreationService(
	woRepo repository.WorkOrderRepository,
	wosRepo repository.WorkOrderServiceRepository,
	wsRepo repository.WorkshopServiceRepository,
	supplyRepo repository.SupplyRepository,
) WorkOrderCreationService {
	return &workOrderCreationService{
		woRepo:     woRepo,
		wosRepo:    wosRepo,
		wsRepo:     wsRepo,
		supplyRepo: supplyRepo,
	}
}

func (s *workOrderCreationService) AddServices(ctx context.Context, workOrderID uuid.UUID, items []AddWorkOrderServiceInput) ([]domain.WorkOrderService, error) {
	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("add services: find work order: %w", err)
	}

	if wo.Status != domain.WorkOrderStatusReceived && wo.Status != domain.WorkOrderStatusInDiagnosis {
		return nil, ErrWorkOrderInvalidStatusForItems
	}

	batch := make([]*domain.WorkOrderService, 0, len(items))
	for _, input := range items {
		ws, err := s.wsRepo.FindByID(ctx, input.ServiceID)
		if err != nil {
			return nil, fmt.Errorf("add services: find service %s: %w", input.ServiceID, err)
		}
		if !ws.Active {
			return nil, ErrWorkshopServiceInactive
		}

		estimatedTime := ws.EstimatedTimeMinutes
		if input.EstimatedTimeMinutes != nil {
			estimatedTime = *input.EstimatedTimeMinutes
		}

		batch = append(batch, &domain.WorkOrderService{
			WorkOrderID:                         workOrderID,
			ServiceID:                           ws.ID,
			ServiceTitleSnapshot:                ws.Title,
			ServiceDescriptionSnapshot:          &ws.Description,
			ServicePriceCentsSnapshot:           ws.PriceCents,
			ServiceEstimatedTimeMinutesSnapshot: estimatedTime,
			ApprovalStatus:                      domain.WorkOrderServiceApprovalPending,
			Status:                              domain.WorkOrderServiceStatusPending,
		})
	}

	created, err := s.wosRepo.CreateBatch(ctx, batch)
	if err != nil {
		return nil, fmt.Errorf("add services: create batch: %w", err)
	}

	if wo.Status == domain.WorkOrderStatusReceived {
		wo.Status = domain.WorkOrderStatusInDiagnosis
		if _, err := s.woRepo.Update(ctx, wo); err != nil {
			return nil, fmt.Errorf("add services: update work order status: %w", err)
		}
	}

	result := make([]domain.WorkOrderService, len(created))
	for i, item := range created {
		result[i] = *item
	}
	return result, nil
}

func (s *workOrderCreationService) RemoveService(ctx context.Context, workOrderID, wosID uuid.UUID) error {
	wos, err := s.wosRepo.FindByID(ctx, wosID)
	if err != nil {
		return fmt.Errorf("remove service: find: %w", err)
	}

	if wos.WorkOrderID != workOrderID {
		return ErrWorkOrderServiceOwnership
	}

	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("remove service: find work order: %w", err)
	}

	if wo.Status == domain.WorkOrderStatusFinished ||
		wo.Status == domain.WorkOrderStatusDelivered ||
		wo.Status == domain.WorkOrderStatusCanceled {
		return ErrWorkOrderInvalidStatusForItems
	}

	if err := s.wosRepo.DeleteByID(ctx, wosID); err != nil {
		return fmt.Errorf("remove service: delete: %w", err)
	}

	return nil
}

func (s *workOrderCreationService) AddSupplies(ctx context.Context, workOrderID, wosID uuid.UUID, items []AddWorkOrderSupplyInput) ([]domain.WorkOrderServiceSupply, error) {
	wos, err := s.wosRepo.FindByID(ctx, wosID)
	if err != nil {
		return nil, fmt.Errorf("add supplies: find work order service: %w", err)
	}
	if wos.WorkOrderID != workOrderID {
		return nil, ErrWorkOrderServiceOwnership
	}

	batch := make([]*domain.WorkOrderServiceSupply, 0, len(items))
	for _, input := range items {
		supply, err := s.supplyRepo.FindByID(ctx, input.SupplyID)
		if err != nil {
			return nil, fmt.Errorf("add supplies: find supply %s: %w", input.SupplyID, err)
		}

		batch = append(batch, &domain.WorkOrderServiceSupply{
			WorkOrderServiceID:       wosID,
			SupplyID:                 supply.ID,
			SupplyTitleSnapshot:      supply.Title,
			SupplyPriceCentsSnapshot: supply.PriceCents,
			SupplyQuantity:           input.Quantity,
		})
	}

	created, err := s.wosRepo.CreateSupplyBatch(ctx, batch)
	if err != nil {
		return nil, fmt.Errorf("add supplies: create batch: %w", err)
	}

	result := make([]domain.WorkOrderServiceSupply, len(created))
	for i, item := range created {
		result[i] = *item
	}
	return result, nil
}
