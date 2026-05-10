package handler

import (
	"context"
	"errors"
	"testing"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeWorkOrderService struct {
	workOrder *domain.WorkOrder
	err       error
}

func (f *fakeWorkOrderService) Create(context.Context, *domain.WorkOrder) (*domain.WorkOrder, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeWorkOrderService) GetByID(context.Context, uuid.UUID) (*domain.WorkOrder, error) {
	return f.workOrder, f.err
}

func (f *fakeWorkOrderService) GetAll(context.Context) ([]domain.WorkOrder, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeWorkOrderService) GetAllWithFilters(context.Context, domain.WorkOrderListFilters) (*domain.WorkOrderListResponse, error) {
	return nil, errors.New("not implemented")
}

func (f *fakeWorkOrderService) Update(context.Context, *domain.WorkOrder) (*domain.WorkOrder, error) {
	return nil, errors.New("not implemented")
}

type fakeBudgetService struct {
	calls []uuid.UUID
	err   error
}

func (f *fakeBudgetService) GenerateAndSendBudget(_ context.Context, workOrderID uuid.UUID) error {
	f.calls = append(f.calls, workOrderID)
	return f.err
}

type fakeWorkOrderCreationService struct {
	addServicesResult []domain.WorkOrderService
	err               error
}

func (f *fakeWorkOrderCreationService) AddServices(context.Context, uuid.UUID, []service.AddWorkOrderServiceInput) ([]domain.WorkOrderService, error) {
	return f.addServicesResult, f.err
}

func (f *fakeWorkOrderCreationService) AddSupplies(context.Context, uuid.UUID, uuid.UUID, []service.AddWorkOrderSupplyInput) ([]domain.WorkOrderServiceSupply, error) {
	return nil, f.err
}

func (f *fakeWorkOrderCreationService) RemoveSupplyFromService(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) error {
	return f.err
}

func (f *fakeWorkOrderCreationService) RemoveService(context.Context, uuid.UUID, uuid.UUID) error {
	return f.err
}

func (f *fakeWorkOrderCreationService) StartService(context.Context, uuid.UUID, uuid.UUID) error {
	return f.err
}

func (f *fakeWorkOrderCreationService) FinalizeService(context.Context, uuid.UUID, uuid.UUID) error {
	return f.err
}

func TestWorkOrderHandler_AddServices_WhenWaitingApprovalResendsBudget(t *testing.T) {
	app := fiber.New()
	workOrderID := uuid.New()
	workOrderServiceID := uuid.New()
	workshopServiceID := uuid.New()
	budgetSvc := &fakeBudgetService{}

	h := NewWorkOrderHandler(
		&fakeWorkOrderService{workOrder: &domain.WorkOrder{ID: workOrderID, Status: domain.WorkOrderStatusWaitingApproval}},
		budgetSvc,
		&fakeWorkOrderCreationService{
			addServicesResult: []domain.WorkOrderService{
				{ID: workOrderServiceID, WorkOrderID: workOrderID, ServiceID: workshopServiceID},
			},
		},
		nil,
		nil,
	)
	app.Post("/work-orders/:id/services", h.AddServices)

	resp, err := flowPostJSON(app, "/work-orders/"+workOrderID.String()+"/services", []map[string]any{
		{"service_id": workshopServiceID.String()},
	})

	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
	require.Len(t, budgetSvc.calls, 1)
	assert.Equal(t, workOrderID, budgetSvc.calls[0])
}
