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
	TransitionTo(ctx context.Context, workOrderID uuid.UUID, newStatus domain.WorkOrderStatus, changedByUserID *uuid.UUID) (*domain.WorkOrder, error)
	IsValidTransition(from, to domain.WorkOrderStatus) bool
}

type workOrderStatusService struct {
	woRepo      repository.WorkOrderRepository
	wosRepo     repository.WorkOrderServiceRepository
	historyRepo repository.WorkOrderStatusHistoryRepository
}

func NewWorkOrderStatusService(
	woRepo repository.WorkOrderRepository,
	wosRepo repository.WorkOrderServiceRepository,
	historyRepo repository.WorkOrderStatusHistoryRepository,
) WorkOrderStatusService {
	return &workOrderStatusService{
		woRepo:      woRepo,
		wosRepo:     wosRepo,
		historyRepo: historyRepo,
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

func (s *workOrderStatusService) TransitionTo(ctx context.Context, workOrderID uuid.UUID, newStatus domain.WorkOrderStatus, changedByUserID *uuid.UUID) (*domain.WorkOrder, error) {
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

	fromStatus := wo.Status
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

	switch newStatus {
	case domain.WorkOrderStatusInProgress:
		if err := s.wosRepo.MarkAsStartedByWorkOrderID(ctx, workOrderID, now); err != nil {
			return nil, fmt.Errorf("transition: mark services as started: %w", err)
		}
	case domain.WorkOrderStatusFinished:
		if err := s.wosRepo.MarkAsFinishedByWorkOrderID(ctx, workOrderID, now); err != nil {
			return nil, fmt.Errorf("transition: mark services as finished: %w", err)
		}
	}

	history := &domain.WorkOrderStatusHistory{
		ID:              uuid.New(),
		WorkOrderID:     workOrderID,
		FromStatus:      fromStatus,
		ToStatus:        newStatus,
		ChangedByUserID: changedByUserID,
		ChangedAt:       now,
	}
	if err := s.historyRepo.Create(ctx, history); err != nil {
		return nil, fmt.Errorf("transition: record history: %w", err)
	}

	return updated, nil
}
