package service

import (
	"context"
	"fmt"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/google/uuid"
)

type WorkOrderItemService interface {
	ApproveService(ctx context.Context, workOrderServiceID uuid.UUID) error
	RejectService(ctx context.Context, workOrderServiceID uuid.UUID) error
	ApproveAllByWorkOrder(ctx context.Context, workOrderID uuid.UUID) error
	RejectAllByWorkOrder(ctx context.Context, workOrderID uuid.UUID) error
}

type workOrderItemService struct {
	wosRepo   repository.WorkOrderServiceRepository
	woRepo    repository.WorkOrderRepository
	statusSvc WorkOrderStatusService
}

func NewWorkOrderItemService(
	wosRepo repository.WorkOrderServiceRepository,
	woRepo repository.WorkOrderRepository,
	statusSvc WorkOrderStatusService,
) WorkOrderItemService {
	return &workOrderItemService{
		wosRepo:   wosRepo,
		woRepo:    woRepo,
		statusSvc: statusSvc,
	}
}

func (s *workOrderItemService) ApproveService(ctx context.Context, workOrderServiceID uuid.UUID) error {
	wos, err := s.wosRepo.FindByID(ctx, workOrderServiceID)
	if err != nil {
		return fmt.Errorf("approve: find service: %w", err)
	}

	if wos.ApprovalStatus != domain.WorkOrderServiceApprovalPending {
		return nil // idempotent
	}

	if err := s.wosRepo.UpdateApprovalStatus(ctx, workOrderServiceID, domain.WorkOrderServiceApprovalApproved); err != nil {
		return fmt.Errorf("approve: update status: %w", err)
	}

	return s.evaluateWorkOrderCompletion(ctx, wos.WorkOrderID)
}

func (s *workOrderItemService) RejectService(ctx context.Context, workOrderServiceID uuid.UUID) error {
	wos, err := s.wosRepo.FindByID(ctx, workOrderServiceID)
	if err != nil {
		return fmt.Errorf("reject: find service: %w", err)
	}

	if wos.ApprovalStatus != domain.WorkOrderServiceApprovalPending {
		return nil // idempotent
	}

	if err := s.wosRepo.UpdateApprovalStatus(ctx, workOrderServiceID, domain.WorkOrderServiceApprovalRejected); err != nil {
		return fmt.Errorf("reject: update status: %w", err)
	}

	return s.evaluateWorkOrderCompletion(ctx, wos.WorkOrderID)
}

func (s *workOrderItemService) ApproveAllByWorkOrder(ctx context.Context, workOrderID uuid.UUID) error {
	if err := s.wosRepo.UpdateApprovalStatusByWorkOrderID(ctx, workOrderID, domain.WorkOrderServiceApprovalApproved); err != nil {
		return fmt.Errorf("approve all: update status: %w", err)
	}

	return s.evaluateWorkOrderCompletion(ctx, workOrderID)
}

func (s *workOrderItemService) RejectAllByWorkOrder(ctx context.Context, workOrderID uuid.UUID) error {
	if err := s.wosRepo.UpdateApprovalStatusByWorkOrderID(ctx, workOrderID, domain.WorkOrderServiceApprovalRejected); err != nil {
		return fmt.Errorf("reject all: update status: %w", err)
	}

	return s.evaluateWorkOrderCompletion(ctx, workOrderID)
}

func (s *workOrderItemService) evaluateWorkOrderCompletion(ctx context.Context, workOrderID uuid.UUID) error {
	services, err := s.wosRepo.FindByWorkOrderID(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("evaluate: find services: %w", err)
	}

	hasApproved := false
	for _, svc := range services {
		if svc.ApprovalStatus == domain.WorkOrderServiceApprovalPending {
			return nil // still pending decisions
		}
		if svc.ApprovalStatus == domain.WorkOrderServiceApprovalApproved {
			hasApproved = true
		}
	}

	var newStatus domain.WorkOrderStatus
	if hasApproved {
		newStatus = domain.WorkOrderStatusApproved
	} else {
		newStatus = domain.WorkOrderStatusCanceled
	}

	wo, err := s.statusSvc.TransitionTo(ctx, workOrderID, newStatus, nil)
	if err != nil {
		return fmt.Errorf("evaluate: transition status: %w", err)
	}

	approvedTotal, err := s.wosRepo.CalculateApprovedTotalForWorkOrder(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("evaluate: calculate approved total: %w", err)
	}
	wo.TotalEstimatedPriceCents = approvedTotal

	if _, err := s.woRepo.Update(ctx, wo); err != nil {
		return fmt.Errorf("evaluate: update work order: %w", err)
	}

	return nil
}
