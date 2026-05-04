package service

import (
	"context"
	"testing"
	"time"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func makeWOS(woID uuid.UUID, approval domain.WorkOrderServiceApprovalStatus) domain.WorkOrderService {
	return domain.WorkOrderService{
		ID:             uuid.New(),
		WorkOrderID:    woID,
		ApprovalStatus: approval,
		Status:         domain.WorkOrderServiceStatusPending,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

func makeWO(id uuid.UUID, status domain.WorkOrderStatus) *domain.WorkOrder {
	return &domain.WorkOrder{
		ID:        id,
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestRejectAll_AllRejected_SetsCanceled(t *testing.T) {
	// when every service is rejected, work order must become CANCELADA (not FINALIZADA)
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderItemService(wosRepo, woRepo)
	ctx := context.Background()
	woID := uuid.New()

	services := []domain.WorkOrderService{
		makeWOS(woID, domain.WorkOrderServiceApprovalRejected),
		makeWOS(woID, domain.WorkOrderServiceApprovalRejected),
	}
	wo := makeWO(woID, domain.WorkOrderStatusWaitingApproval)

	wosRepo.On("UpdateApprovalStatusByWorkOrderID", ctx, woID, domain.WorkOrderServiceApprovalRejected).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("CalculateApprovedTotalForWorkOrder", ctx, woID).Return(0, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	err := svc.RejectAllByWorkOrder(ctx, woID)
	assert.NoError(t, err)

	call := woRepo.Calls[len(woRepo.Calls)-1]
	updated := call.Arguments[1].(*domain.WorkOrder)
	assert.Equal(t, domain.WorkOrderStatusCanceled, updated.Status)
}

func TestRejectService_LastPending_AllRejected_SetsCanceled(t *testing.T) {
	// when the last pending service is rejected and none are approved, WO becomes CANCELADA
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderItemService(wosRepo, woRepo)
	ctx := context.Background()
	woID := uuid.New()
	wosID := uuid.New()

	pending := &domain.WorkOrderService{
		ID:             wosID,
		WorkOrderID:    woID,
		ApprovalStatus: domain.WorkOrderServiceApprovalPending,
	}
	afterReject := []domain.WorkOrderService{
		makeWOS(woID, domain.WorkOrderServiceApprovalRejected),
		{ID: wosID, WorkOrderID: woID, ApprovalStatus: domain.WorkOrderServiceApprovalRejected},
	}
	wo := makeWO(woID, domain.WorkOrderStatusWaitingApproval)

	wosRepo.On("FindByID", ctx, wosID).Return(pending, nil)
	wosRepo.On("UpdateApprovalStatus", ctx, wosID, domain.WorkOrderServiceApprovalRejected).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(afterReject, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("CalculateApprovedTotalForWorkOrder", ctx, woID).Return(0, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	err := svc.RejectService(ctx, wosID)
	assert.NoError(t, err)

	call := woRepo.Calls[len(woRepo.Calls)-1]
	updated := call.Arguments[1].(*domain.WorkOrder)
	assert.Equal(t, domain.WorkOrderStatusCanceled, updated.Status)
}

func TestApproveAll_AllApproved_SetsApproved(t *testing.T) {
	// existing behaviour: when all approved, WO becomes APROVADO
	wosRepo := new(mockWorkOrderServiceRepo)
	woRepo := new(mockWorkOrderRepo)
	svc := NewWorkOrderItemService(wosRepo, woRepo)
	ctx := context.Background()
	woID := uuid.New()

	services := []domain.WorkOrderService{
		makeWOS(woID, domain.WorkOrderServiceApprovalApproved),
	}
	wo := makeWO(woID, domain.WorkOrderStatusWaitingApproval)

	wosRepo.On("UpdateApprovalStatusByWorkOrderID", ctx, woID, domain.WorkOrderServiceApprovalApproved).Return(nil)
	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(services, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("CalculateApprovedTotalForWorkOrder", ctx, woID).Return(10000, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	err := svc.ApproveAllByWorkOrder(ctx, woID)
	assert.NoError(t, err)

	call := woRepo.Calls[len(woRepo.Calls)-1]
	updated := call.Arguments[1].(*domain.WorkOrder)
	assert.Equal(t, domain.WorkOrderStatusApproved, updated.Status)
}
