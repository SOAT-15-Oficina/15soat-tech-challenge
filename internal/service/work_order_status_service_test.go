package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTransitionTo_ValidTransition_UpdatesStatus(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	svc := NewWorkOrderStatusService(woRepo, wosRepo)
	ctx := context.Background()

	woID := uuid.New()
	wo := &domain.WorkOrder{
		ID:        woID,
		Status:    domain.WorkOrderStatusReceived,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	result, err := svc.TransitionTo(ctx, woID, domain.WorkOrderStatusInDiagnosis)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, domain.WorkOrderStatusInDiagnosis, wo.Status)
}

func TestTransitionTo_WaitingApproval_GeneratesBudget(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	budgetSvc := new(mockBudgetServiceUseCase)
	svc := NewWorkOrderStatusService(woRepo, wosRepo, WithBudgetGeneration(budgetSvc))
	ctx := context.Background()

	woID := uuid.New()
	wo := &domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusInDiagnosis}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)
	budgetSvc.On("GenerateAndSendBudget", ctx, woID).Return(nil)

	result, err := svc.TransitionTo(ctx, woID, domain.WorkOrderStatusWaitingApproval)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	budgetSvc.AssertCalled(t, "GenerateAndSendBudget", ctx, woID)
}

func TestTransitionTo_WaitingApproval_BudgetGenerationFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	budgetSvc := new(mockBudgetServiceUseCase)
	svc := NewWorkOrderStatusService(woRepo, wosRepo, WithBudgetGeneration(budgetSvc))
	ctx := context.Background()

	woID := uuid.New()
	wo := &domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusInDiagnosis}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)
	budgetSvc.On("GenerateAndSendBudget", ctx, woID).Return(errors.New("smtp error"))

	result, err := svc.TransitionTo(ctx, woID, domain.WorkOrderStatusWaitingApproval)
	assert.Error(t, err)
	assert.Nil(t, result)
}

func TestTransitionTo_InvalidTransition_ReturnsError(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	svc := NewWorkOrderStatusService(woRepo, wosRepo)
	ctx := context.Background()

	woID := uuid.New()
	wo := &domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusDelivered}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	_, err := svc.TransitionTo(ctx, woID, domain.WorkOrderStatusReceived)
	assert.ErrorIs(t, err, ErrInvalidStatusTransition)
	woRepo.AssertNotCalled(t, "Update")
}

func TestTransitionTo_SameStatus_Idempotent(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	svc := NewWorkOrderStatusService(woRepo, wosRepo)
	ctx := context.Background()

	woID := uuid.New()
	wo := &domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusReceived}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	result, err := svc.TransitionTo(ctx, woID, domain.WorkOrderStatusReceived)
	assert.NoError(t, err)
	assert.Equal(t, wo, result)
	woRepo.AssertNotCalled(t, "Update")
}

func TestTransitionTo_SetsTimestamps(t *testing.T) {
	tests := []struct {
		name         string
		from         domain.WorkOrderStatus
		to           domain.WorkOrderStatus
		checkFunc    func(t *testing.T, wo *domain.WorkOrder)
		mockWosSetup func(wosRepo *mockWorkOrderServiceRepo, woID uuid.UUID)
	}{
		{
			name: "approved sets approved_at",
			from: domain.WorkOrderStatusWaitingApproval,
			to:   domain.WorkOrderStatusApproved,
			checkFunc: func(t *testing.T, wo *domain.WorkOrder) {
				assert.NotNil(t, wo.ApprovedAt)
			},
		},
		{
			name: "in_progress sets started_at",
			from: domain.WorkOrderStatusApproved,
			to:   domain.WorkOrderStatusInProgress,
			checkFunc: func(t *testing.T, wo *domain.WorkOrder) {
				assert.NotNil(t, wo.StartedAt)
			},
		},
		{
			name: "finished sets finished_at",
			from: domain.WorkOrderStatusInProgress,
			to:   domain.WorkOrderStatusFinished,
			checkFunc: func(t *testing.T, wo *domain.WorkOrder) {
				assert.NotNil(t, wo.FinishedAt)
			},
		},
		{
			name: "delivered sets delivered_at",
			from: domain.WorkOrderStatusFinished,
			to:   domain.WorkOrderStatusDelivered,
			checkFunc: func(t *testing.T, wo *domain.WorkOrder) {
				assert.NotNil(t, wo.DeliveredAt)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			woRepo := new(mockWorkOrderRepo)
			wosRepo := new(mockWorkOrderServiceRepo)
			svc := NewWorkOrderStatusService(woRepo, wosRepo)
			ctx := context.Background()

			woID := uuid.New()
			wo := &domain.WorkOrder{ID: woID, Status: tt.from}

			woRepo.On("FindByID", ctx, woID).Return(wo, nil)
			woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)
			if tt.mockWosSetup != nil {
				tt.mockWosSetup(wosRepo, woID)
			}

			_, err := svc.TransitionTo(ctx, woID, tt.to)
			assert.NoError(t, err)

			updateCall := woRepo.Calls[len(woRepo.Calls)-1]
			updated := updateCall.Arguments[1].(*domain.WorkOrder)
			tt.checkFunc(t, updated)
		})
	}
}

func TestTransitionTo_InProgress_DoesNotBulkMarkServices(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	svc := NewWorkOrderStatusService(woRepo, wosRepo)
	ctx := context.Background()

	woID := uuid.New()
	wo := &domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusApproved}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	_, err := svc.TransitionTo(ctx, woID, domain.WorkOrderStatusInProgress)
	assert.NoError(t, err)
	wosRepo.AssertNotCalled(t, "MarkAsStartedByWorkOrderID")
}

func TestTransitionTo_Finished_DoesNotBulkMarkServices(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	svc := NewWorkOrderStatusService(woRepo, wosRepo)
	ctx := context.Background()

	woID := uuid.New()
	wo := &domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusInProgress}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	_, err := svc.TransitionTo(ctx, woID, domain.WorkOrderStatusFinished)
	assert.NoError(t, err)
	wosRepo.AssertNotCalled(t, "MarkAsFinishedByWorkOrderID")
}

func TestIsValidTransition_AllowedTransitions(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	svc := NewWorkOrderStatusService(woRepo, wosRepo)

	valid := []struct{ from, to domain.WorkOrderStatus }{
		{domain.WorkOrderStatusReceived, domain.WorkOrderStatusInDiagnosis},
		{domain.WorkOrderStatusReceived, domain.WorkOrderStatusCanceled},
		{domain.WorkOrderStatusInDiagnosis, domain.WorkOrderStatusWaitingApproval},
		{domain.WorkOrderStatusWaitingApproval, domain.WorkOrderStatusApproved},
		{domain.WorkOrderStatusWaitingApproval, domain.WorkOrderStatusCanceled},
		{domain.WorkOrderStatusApproved, domain.WorkOrderStatusInProgress},
		{domain.WorkOrderStatusInProgress, domain.WorkOrderStatusFinished},
		{domain.WorkOrderStatusFinished, domain.WorkOrderStatusDelivered},
	}
	for _, tc := range valid {
		assert.True(t, svc.IsValidTransition(tc.from, tc.to), "%s -> %s should be valid", tc.from, tc.to)
	}

	invalid := []struct{ from, to domain.WorkOrderStatus }{
		{domain.WorkOrderStatusDelivered, domain.WorkOrderStatusReceived},
		{domain.WorkOrderStatusReceived, domain.WorkOrderStatusFinished},
		{domain.WorkOrderStatusCanceled, domain.WorkOrderStatusReceived},
		{domain.WorkOrderStatusFinished, domain.WorkOrderStatusInProgress},
	}
	for _, tc := range invalid {
		assert.False(t, svc.IsValidTransition(tc.from, tc.to), "%s -> %s should be invalid", tc.from, tc.to)
	}
}
