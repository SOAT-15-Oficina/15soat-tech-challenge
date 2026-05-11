package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/google/uuid"
)

var (
	ErrWorkOrderInvalidStatusForItems = errors.New("work order status does not allow adding services")
	ErrWorkshopServiceInactive        = errors.New("workshop service is inactive")
	ErrWorkOrderServiceOwnership      = errors.New("work order service does not belong to this work order")
	ErrWorkOrderNotInProgress         = errors.New("work order must be in EM_EXECUCAO status")
	ErrServiceNotPending              = errors.New("service must be PENDENTE to start")
	ErrServiceNotApproved             = errors.New("service must be approved to start")
	ErrServiceNotInProgress           = errors.New("service must be in EM_EXECUCAO status to finalize")
	ErrInsufficientStock              = errors.New("insufficient stock for service supplies")
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
	RemoveSupplyFromService(ctx context.Context, workOrderID, wosID, supplyID uuid.UUID) error
	RemoveService(ctx context.Context, workOrderID, wosID uuid.UUID) error
	// StartService returns (delayAdded bool, err error). delayAdded=true means stock
	// was insufficient and a 2-day delay was added to the work order's expected delivery.
	StartService(ctx context.Context, workOrderID, wosID uuid.UUID) (bool, error)
	FinalizeService(ctx context.Context, workOrderID, wosID uuid.UUID) error
}

type workOrderCreationService struct {
	woRepo     repository.WorkOrderRepository
	wosRepo    repository.WorkOrderServiceRepository
	wsRepo     repository.WorkshopServiceRepository
	supplyRepo repository.SupplyRepository
	statusSvc  WorkOrderStatusService
}

func NewWorkOrderCreationService(
	woRepo repository.WorkOrderRepository,
	wosRepo repository.WorkOrderServiceRepository,
	wsRepo repository.WorkshopServiceRepository,
	supplyRepo repository.SupplyRepository,
	statusSvc WorkOrderStatusService,
) WorkOrderCreationService {
	return &workOrderCreationService{
		woRepo:     woRepo,
		wosRepo:    wosRepo,
		wsRepo:     wsRepo,
		supplyRepo: supplyRepo,
		statusSvc:  statusSvc,
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
		if _, err := s.statusSvc.TransitionTo(ctx, workOrderID, domain.WorkOrderStatusInDiagnosis); err != nil {
			return nil, fmt.Errorf("add services: transition status: %w", err)
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

	if err := s.wosRepo.DeleteSuppliesByWorkOrderServiceID(ctx, wosID); err != nil {
		return fmt.Errorf("remove service: delete supplies: %w", err)
	}

	if err := s.wosRepo.DeleteByID(ctx, wosID); err != nil {
		return fmt.Errorf("remove service: delete: %w", err)
	}

	return nil
}

func (s *workOrderCreationService) RemoveSupplyFromService(ctx context.Context, workOrderID, wosID, supplyID uuid.UUID) error {
	wos, err := s.wosRepo.FindByID(ctx, wosID)
	if err != nil {
		return fmt.Errorf("remove supply: find work order service: %w", err)
	}
	if wos.WorkOrderID != workOrderID {
		return ErrWorkOrderServiceOwnership
	}

	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("remove supply: find work order: %w", err)
	}

	if wo.Status == domain.WorkOrderStatusFinished ||
		wo.Status == domain.WorkOrderStatusDelivered ||
		wo.Status == domain.WorkOrderStatusCanceled {
		return ErrWorkOrderInvalidStatusForItems
	}

	if err := s.wosRepo.DeleteSupplyForWorkOrderService(ctx, wosID, supplyID); err != nil {
		return fmt.Errorf("remove supply: delete: %w", err)
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

func (s *workOrderCreationService) StartService(ctx context.Context, workOrderID, wosID uuid.UUID) (bool, error) {
	wos, err := s.wosRepo.FindByID(ctx, wosID)
	if err != nil {
		return false, fmt.Errorf("start service: find: %w", err)
	}
	if wos.WorkOrderID != workOrderID {
		return false, ErrWorkOrderServiceOwnership
	}

	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return false, fmt.Errorf("start service: find work order: %w", err)
	}
	if wo.Status != domain.WorkOrderStatusInProgress {
		return false, ErrWorkOrderNotInProgress
	}
	if wos.ApprovalStatus != domain.WorkOrderServiceApprovalApproved {
		return false, ErrServiceNotApproved
	}
	if wos.Status != domain.WorkOrderServiceStatusPending {
		return false, ErrServiceNotPending
	}

	delayAdded := false
	hasShortage, err := s.wosRepo.HasSupplyShortagesForService(ctx, wosID)
	if err != nil {
		return false, fmt.Errorf("start service: check stock: %w", err)
	}
	if hasShortage {
		if err := s.woRepo.AddDeliveryDelay(ctx, workOrderID, 2); err != nil {
			return false, fmt.Errorf("start service: add delivery delay: %w", err)
		}
		delayAdded = true
	}

	if err := s.wosRepo.MarkServiceAsStarted(ctx, wosID, time.Now()); err != nil {
		return false, fmt.Errorf("start service: update: %w", err)
	}
	return delayAdded, nil
}

func (s *workOrderCreationService) FinalizeService(ctx context.Context, workOrderID, wosID uuid.UUID) error {
	wos, err := s.wosRepo.FindByID(ctx, wosID)
	if err != nil {
		return fmt.Errorf("finalize service: find: %w", err)
	}
	if wos.WorkOrderID != workOrderID {
		return ErrWorkOrderServiceOwnership
	}

	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("finalize service: find work order: %w", err)
	}
	if wo.Status != domain.WorkOrderStatusInProgress {
		return ErrWorkOrderNotInProgress
	}
	if wos.Status != domain.WorkOrderServiceStatusInProgress {
		return ErrServiceNotInProgress
	}

	if err := s.wosRepo.MarkServiceAsFinished(ctx, wosID, time.Now()); err != nil {
		return fmt.Errorf("finalize service: update: %w", err)
	}

	// Decrement stock for supplies used in this service
	if err := s.supplyRepo.DecrementStockForService(ctx, wosID); err != nil {
		return fmt.Errorf("finalize service: decrement stock: %w", err)
	}

	// Check if all approved services are now finalized → auto-finalize WO
	services, err := s.wosRepo.FindByWorkOrderID(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("finalize service: check completion: %w", err)
	}

	allFinished := true
	for _, svc := range services {
		if svc.ApprovalStatus == domain.WorkOrderServiceApprovalApproved &&
			svc.Status != domain.WorkOrderServiceStatusFinished {
			allFinished = false
			break
		}
	}

	if allFinished {
		if _, err := s.statusSvc.TransitionTo(ctx, workOrderID, domain.WorkOrderStatusFinished); err != nil {
			return fmt.Errorf("finalize service: auto-transition: %w", err)
		}
	}

	return nil
}
