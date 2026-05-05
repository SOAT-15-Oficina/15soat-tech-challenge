package service

import (
	"context"
	"testing"
	"time"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
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
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusInDiagnosis, (*uuid.UUID)(nil)).Return(wo, nil)

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
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
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
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
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
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
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
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
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
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
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

func TestAddServices_WorkOrderRecebida_TransitionsToEmDiagnostico(t *testing.T) {
	// when WO is RECEBIDA and first service is added, status must become EM_DIAGNOSTICO
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wsID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusReceived)
	ws := activeWorkshopService(wsID)

	created := []*domain.WorkOrderService{
		{ID: uuid.New(), WorkOrderID: woID, ServiceID: wsID},
	}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wsRepo.On("FindByID", ctx, wsID).Return(ws, nil)
	wosRepo.On("CreateBatch", ctx, mock.AnythingOfType("[]*domain.WorkOrderService")).Return(created, nil)
	statusSvc.On("TransitionTo", ctx, woID, domain.WorkOrderStatusInDiagnosis, (*uuid.UUID)(nil)).Return(wo, nil)

	_, err := svc.AddServices(ctx, woID, []AddWorkOrderServiceInput{{ServiceID: wsID}})
	assert.NoError(t, err)
	statusSvc.AssertCalled(t, "TransitionTo", ctx, woID, domain.WorkOrderStatusInDiagnosis, (*uuid.UUID)(nil))
}

func TestAddServices_WorkOrderEmDiagnostico_DoesNotChangeStatus(t *testing.T) {
	// when WO is already EM_DIAGNOSTICO, status must not change after adding service
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wsID := uuid.New()
	wo := openWO(woID, domain.WorkOrderStatusInDiagnosis)
	ws := activeWorkshopService(wsID)

	created := []*domain.WorkOrderService{
		{ID: uuid.New(), WorkOrderID: woID, ServiceID: wsID},
	}

	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wsRepo.On("FindByID", ctx, wsID).Return(ws, nil)
	wosRepo.On("CreateBatch", ctx, mock.AnythingOfType("[]*domain.WorkOrderService")).Return(created, nil)

	_, err := svc.AddServices(ctx, woID, []AddWorkOrderServiceInput{{ServiceID: wsID}})
	assert.NoError(t, err)
	statusSvc.AssertNotCalled(t, "TransitionTo")
}

func TestRemoveService_Valid_DeletesCalled(t *testing.T) {
	// happy path: valid ownership and non-final WO status triggers DeleteByID
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	wo := openWO(woID, domain.WorkOrderStatusInDiagnosis)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("DeleteSuppliesByWorkOrderServiceID", ctx, wosID).Return(nil)
	wosRepo.On("DeleteByID", ctx, wosID).Return(nil)

	err := svc.RemoveService(ctx, woID, wosID)
	assert.NoError(t, err)
	wosRepo.AssertCalled(t, "DeleteSuppliesByWorkOrderServiceID", ctx, wosID)
	wosRepo.AssertCalled(t, "DeleteByID", ctx, wosID)
}

func TestRemoveService_WosNotFound_ReturnsNotFoundError(t *testing.T) {
	// when wosID does not exist, must propagate pgx.ErrNoRows
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	wosID := uuid.New()
	wosRepo.On("FindByID", ctx, wosID).Return(nil, pgx.ErrNoRows)

	err := svc.RemoveService(ctx, uuid.New(), wosID)
	assert.ErrorIs(t, err, pgx.ErrNoRows)
	wosRepo.AssertNotCalled(t, "DeleteByID")
}

func TestRemoveService_WosWrongWorkOrder_ReturnsOwnershipError(t *testing.T) {
	// wosID exists but belongs to a different work order
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: uuid.New()}
	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)

	err := svc.RemoveService(ctx, woID, wosID)
	assert.ErrorIs(t, err, ErrWorkOrderServiceOwnership)
	wosRepo.AssertNotCalled(t, "DeleteByID")
}

func TestRemoveService_WorkOrderFinalStatus_ReturnsInvalidStatusError(t *testing.T) {
	// must reject removal when WO is in a final status
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	statusSvc := new(mockStatusService)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	wo := openWO(woID, domain.WorkOrderStatusFinished)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	err := svc.RemoveService(ctx, woID, wosID)
	assert.ErrorIs(t, err, ErrWorkOrderInvalidStatusForItems)
	wosRepo.AssertNotCalled(t, "DeleteByID")
}

func TestRemoveSupplyFromService_Valid_DeletesRow(t *testing.T) {
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
	wo := openWO(woID, domain.WorkOrderStatusInDiagnosis)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	wosRepo.On("DeleteSupplyForWorkOrderService", ctx, wosID, supplyID).Return(nil)

	err := svc.RemoveSupplyFromService(ctx, woID, wosID, supplyID)
	assert.NoError(t, err)
	wosRepo.AssertCalled(t, "DeleteSupplyForWorkOrderService", ctx, wosID, supplyID)
}

func TestRemoveSupplyFromService_WosWrongWorkOrder_ReturnsOwnershipError(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: uuid.New()}
	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)

	err := svc.RemoveSupplyFromService(ctx, woID, wosID, uuid.New())
	assert.ErrorIs(t, err, ErrWorkOrderServiceOwnership)
	wosRepo.AssertNotCalled(t, "DeleteSupplyForWorkOrderService")
}

func TestRemoveSupplyFromService_WorkOrderFinalStatus_ReturnsInvalidStatusError(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	wsRepo := new(mockWorkshopServiceRepo)
	supplyRepo := new(mockSupplyRepo)
	svc := NewWorkOrderCreationService(woRepo, wosRepo, wsRepo, supplyRepo)
	ctx := context.Background()

	woID := uuid.New()
	wosID := uuid.New()
	wos := &domain.WorkOrderService{ID: wosID, WorkOrderID: woID}
	wo := openWO(woID, domain.WorkOrderStatusDelivered)

	wosRepo.On("FindByID", ctx, wosID).Return(wos, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)

	err := svc.RemoveSupplyFromService(ctx, woID, wosID, uuid.New())
	assert.ErrorIs(t, err, ErrWorkOrderInvalidStatusForItems)
	wosRepo.AssertNotCalled(t, "DeleteSupplyForWorkOrderService")
}
