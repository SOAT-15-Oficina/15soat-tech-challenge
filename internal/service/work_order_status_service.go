package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/application"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/google/uuid"
)

var ErrInvalidStatusTransition = errors.New("invalid status transition")

var allowedTransitions = map[domain.WorkOrderStatus][]domain.WorkOrderStatus{
	domain.WorkOrderStatusReceived:        {domain.WorkOrderStatusInDiagnosis, domain.WorkOrderStatusCanceled},
	domain.WorkOrderStatusInDiagnosis:     {domain.WorkOrderStatusWaitingApproval, domain.WorkOrderStatusCanceled},
	domain.WorkOrderStatusWaitingApproval: {domain.WorkOrderStatusApproved, domain.WorkOrderStatusCanceled},
	domain.WorkOrderStatusApproved:        {domain.WorkOrderStatusInProgress},
	domain.WorkOrderStatusInProgress:      {domain.WorkOrderStatusFinished},
	domain.WorkOrderStatusFinished:        {domain.WorkOrderStatusDelivered},
	domain.WorkOrderStatusDelivered:       {},
	domain.WorkOrderStatusCanceled:        {},
}

type WorkOrderStatusService interface {
	TransitionTo(ctx context.Context, workOrderID uuid.UUID, newStatus domain.WorkOrderStatus) (*domain.WorkOrder, error)
	IsValidTransition(from, to domain.WorkOrderStatus) bool
}

type workOrderStatusService struct {
	woRepo  application.WorkOrderRepository
	wosRepo application.WorkOrderServiceRepository
	budget  BudgetService
}

func NewWorkOrderStatusService(
	woRepo application.WorkOrderRepository,
	wosRepo application.WorkOrderServiceRepository,
	opts ...WorkOrderStatusServiceOption,
) WorkOrderStatusService {
	svc := &workOrderStatusService{
		woRepo:  woRepo,
		wosRepo: wosRepo,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

type WorkOrderStatusServiceOption func(*workOrderStatusService)

func WithBudgetGeneration(budget BudgetService) WorkOrderStatusServiceOption {
	return func(s *workOrderStatusService) {
		s.budget = budget
	}
}

func (s *workOrderStatusService) IsValidTransition(from, to domain.WorkOrderStatus) bool {
	allowed, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	for _, status := range allowed {
		if status == to {
			return true
		}
	}
	return false
}

func (s *workOrderStatusService) TransitionTo(ctx context.Context, workOrderID uuid.UUID, newStatus domain.WorkOrderStatus) (*domain.WorkOrder, error) {
	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return nil, fmt.Errorf("transition: find work order: %w", err)
	}

	if wo.Status == newStatus {
		return wo, nil
	}

	if !s.IsValidTransition(wo.Status, newStatus) {
		return nil, fmt.Errorf("%w: %s -> %s", ErrInvalidStatusTransition, wo.Status, newStatus)
	}

	wo.Status = newStatus
	now := time.Now()
	wo.UpdatedAt = now

	switch newStatus {
	case domain.WorkOrderStatusApproved:
		wo.ApprovedAt = &now
	case domain.WorkOrderStatusInProgress:
		wo.StartedAt = &now
	case domain.WorkOrderStatusFinished:
		wo.FinishedAt = &now
	case domain.WorkOrderStatusDelivered:
		wo.DeliveredAt = &now
	}

	updated, err := s.woRepo.Update(ctx, wo)
	if err != nil {
		return nil, fmt.Errorf("transition: update work order: %w", err)
	}

	if updated.Status == domain.WorkOrderStatusWaitingApproval && s.budget != nil {
		if err := s.budget.GenerateAndSendBudget(ctx, updated.ID); err != nil {
			return nil, fmt.Errorf("transition: generate budget: %w", err)
		}
	}

	return updated, nil
}
