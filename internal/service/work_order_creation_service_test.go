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

func activeWorkshopService(id uuid.UUID) *domain.WorkshopService {
	return &domain.WorkshopService{
		ID:                   id,
		Title:                "Troca de óleo",
		Description:          "Troca completa",
		PriceCents:           8000,
		EstimatedTimeMinutes: 30,
		Active:               true,
	}
}

func inactiveWorkshopService(id uuid.UUID) *domain.WorkshopService {
	ws := activeWorkshopService(id)
	ws.Active = false
	return ws
}

func activeSupply(id uuid.UUID) *domain.Supply {
	return &domain.Supply{
		ID:         id,
		Title:      "Filtro de óleo",
		PriceCents: 3500,
		Active:     true,
	}
}

func openWO(id uuid.UUID, status domain.WorkOrderStatus) *domain.WorkOrder {
	return &domain.WorkOrder{
		ID:        id,
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

func TestAddServices_ValidInput_CreatesItems(t *testing.T) {
	// should create work order service records with snapshots from catalog
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo)
	ctx := context.Background()

	woID := uuid.New()
	wsID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusReceived)
	ws := activeWorkshopService(wsID)

	created := []*domain.WorkOrderService{
		{
			ID:                                  uuid.New(),
			WorkOrderID:                         woID,
			ServiceID:                           wsID,
			ServiceTitleSnapshot:                ws.Title,
			ServicePriceCentsSnapshot:           ws.PriceCents,
			ServiceEstimatedTimeMinutesSnapshot: ws.EstimatedTimeMinutes,
			ApprovalStatus:                      domain.WorkOrderServiceApprovalPending,
			Status:                              domain.WorkOrderServiceStatusPending,
		},
	}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wsRepo.On("FindByID", ctx, wsID).Return(ws, nil)
	wosRepo.On("CreateBatch", ctx, mock.AnythingOfType("[]*domain.WorkOrderService")).Return(created, nil)

	items := []AddWorkOrderServiceInput{{ServiceID: wsID}}
	result, err := svc.AddServices(ctx, woID, items)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, ws.Title, result[0].ServiceTitleSnapshot)
	assert.Equal(t, ws.PriceCents, result[0].ServicePriceCentsSnapshot)
	assert.Equal(t, domain.WorkOrderServiceApprovalPending, result[0].ApprovalStatus)
}

func TestAddServices_InvalidStatus_ReturnsError(t *testing.T) {
	// should reject when work order is not in RECEBIDA or EM_DIAGNOSTICO
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo)
	ctx := context.Background()

	woID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusApproved)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	items := []AddWorkOrderServiceInput{{ServiceID: uuid.New()}}
	result, err := svc.AddServices(ctx, woID, items)

	assert.ErrorIs(t, err, ErrWorkOrderInvalidStatusForItems)
	assert.Nil(t, result)
	wosRepo.AssertNotCalled(t, "CreateBatch")
}

func TestAddServices_InactiveService_ReturnsError(t *testing.T) {
	// should reject when a service in the catalog is inactive
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo)
	ctx := context.Background()

	woID := uuid.New()
	wsID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusReceived)
	ws := inactiveWorkshopService(wsID)

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wsRepo.On("FindByID", ctx, wsID).Return(ws, nil)

	items := []AddWorkOrderServiceInput{{ServiceID: wsID}}
	result, err := svc.AddServices(ctx, woID, items)

	assert.ErrorIs(t, err, ErrWorkshopServiceInactive)
	assert.Nil(t, result)
	wosRepo.AssertNotCalled(t, "CreateBatch")
}

func TestAddServices_OptionalEstimatedTime_UsesCustomValue(t *testing.T) {
	// when estimated_time_minutes is provided, it overrides the catalog value
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo)
	ctx := context.Background()

	woID := uuid.New()
	wsID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusInDiagnosis)
	ws := activeWorkshopService(wsID)
	customTime := 90

	created := []*domain.WorkOrderService{
		{
			ID:                                  uuid.New(),
			WorkOrderID:                         woID,
			ServiceID:                           wsID,
			ServiceEstimatedTimeMinutesSnapshot: customTime,
		},
	}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wsRepo.On("FindByID", ctx, wsID).Return(ws, nil)
	wosRepo.On("CreateBatch", ctx, mock.AnythingOfType("[]*domain.WorkOrderService")).Return(created, nil)

	items := []AddWorkOrderServiceInput{{ServiceID: wsID, EstimatedTimeMinutes: &customTime}}
	_, err := svc.AddServices(ctx, woID, items)
	assert.NoError(t, err)

	call := wosRepo.Calls[0]
	submitted := call.Arguments[1].([]*domain.WorkOrderService)
	assert.Equal(t, customTime, submitted[0].ServiceEstimatedTimeMinutesSnapshot)
}

func TestAddSupplies_ValidInput_CreatesItems(t *testing.T) {
	// should create supply records with snapshots
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	supplyID := uuid.New()

	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	supply := activeSupply(supplyID)

	created := []*domain.WorkOrderServiceSupply{
		{
			ID:                       uuid.New(),
			WorkOrderServiceID:       wosID,
			SupplyID:                 supplyID,
			SupplyTitleSnapshot:      supply.Title,
			SupplyPriceCentsSnapshot: supply.PriceCents,
			SupplyQuantity:           2,
		},
	}

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	supplyRepo.On("FindByID", ctx, supplyID).Return(supply, nil)
	wosRepo.On("CreateSupplyBatch", ctx, mock.AnythingOfType("[]*domain.WorkOrderServiceSupply")).Return(created, nil)

	items := []AddWorkOrderSupplyInput{{SupplyID: supplyID, Quantity: 2}}
	result, err := svc.AddSupplies(ctx, woID, wosID, items)

	assert.NoError(t, err)
	assert.Len(t, result, 1)
	assert.Equal(t, supply.Title, result[0].SupplyTitleSnapshot)
	assert.Equal(t, 2, result[0].SupplyQuantity)
}

func TestAddSupplies_WosNotBelongingToWorkOrder_ReturnsError(t *testing.T) {
	// must reject when wosID belongs to a different work order
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo)
	ctx := context.Background()

	woID := uuid.New()
	otherWOID := uuid.New()
	wosID := uuid.New()

	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: otherWOID}
	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)

	items := []AddWorkOrderSupplyInput{{SupplyID: uuid.New(), Quantity: 1}}
	result, err := svc.AddSupplies(ctx, woID, wosID, items)

	assert.ErrorIs(t, err, ErrWorkOrderServiceOwnership)
	assert.Nil(t, result)
	wosRepo.AssertNotCalled(t, "CreateSupplyBatch")
}
