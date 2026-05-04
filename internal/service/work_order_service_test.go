package service

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

var codePattern = regexp.MustCompile(`^OS-\d{8}-[0-9A-F]{4}$`)

func newWOInput(customerID, vehicleID, userID uuid.UUID) *domain.WorkOrder {
	return &domain.WorkOrder{
		Title:          "Revisão geral",
		CustomerID:     customerID,
		VehicleID:      vehicleID,
		OpenedByUserID: userID,
	}
}

func savedWO(id, customerID, vehicleID, userID uuid.UUID) *domain.WorkOrder {
	wo := newWOInput(customerID, vehicleID, userID)
	wo.ID = id
	wo.Code = "OS-20260504-AB12"
	wo.Status = domain.WorkOrderStatusReceived
	return wo
}

func TestCreate_AutoGeneratesCode(t *testing.T) {
	// code must be generated when not provided, matching OS-YYYYMMDD-XXXX
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	customerID := uuid.New()
	vehicleID := uuid.New()
	userID := uuid.New()
	input := newWOInput(customerID, vehicleID, userID)

	vehicle := &domain.Vehicle{ID: vehicleID, CustomerID: customerID}
	result := savedWO(uuid.New(), customerID, vehicleID, userID)

	vehicleRepo.On("FindByID", ctx, vehicleID).Return(vehicle, nil)
	woRepo.On("Create", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(result, nil)

	out, err := svc.Create(ctx, input)
	assert.NoError(t, err)
	assert.NotNil(t, out)

	// inspect the code that was set on the input before repo.Create was called
	call := woRepo.Calls[0]
	submitted := call.Arguments[1].(*domain.WorkOrder)
	assert.Regexp(t, codePattern, submitted.Code)
}

func TestCreate_VehicleNotBelongingToCustomer_ReturnsError(t *testing.T) {
	// must reject when vehicle.CustomerID != workOrder.CustomerID
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	customerID := uuid.New()
	otherCustomerID := uuid.New()
	vehicleID := uuid.New()
	userID := uuid.New()
	input := newWOInput(customerID, vehicleID, userID)

	vehicle := &domain.Vehicle{ID: vehicleID, CustomerID: otherCustomerID}
	vehicleRepo.On("FindByID", ctx, vehicleID).Return(vehicle, nil)

	out, err := svc.Create(ctx, input)
	assert.ErrorIs(t, err, ErrVehicleNotBelongingToCustomer)
	assert.Nil(t, out)
	woRepo.AssertNotCalled(t, "Create")
}

func TestCreate_VehicleNotFound_ReturnsError(t *testing.T) {
	// must propagate error when vehicle does not exist
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	customerID := uuid.New()
	vehicleID := uuid.New()
	userID := uuid.New()
	input := newWOInput(customerID, vehicleID, userID)

	vehicleRepo.On("FindByID", ctx, vehicleID).Return(nil, errors.New("not found"))

	out, err := svc.Create(ctx, input)
	assert.Error(t, err)
	assert.Nil(t, out)
	woRepo.AssertNotCalled(t, "Create")
}

func TestCreate_MissingTitle_ReturnsError(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	input := &domain.WorkOrder{
		CustomerID:     uuid.New(),
		VehicleID:      uuid.New(),
		OpenedByUserID: uuid.New(),
	}

	out, err := svc.Create(ctx, input)
	assert.Error(t, err)
	assert.Nil(t, out)
}

func TestCreate_MissingCustomerID_ReturnsError(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	vehicleRepo := new(mockVehicleRepo)
	svc := NewWorkOrderService(woRepo, vehicleRepo)
	ctx := context.Background()

	input := &domain.WorkOrder{
		Title:          "Revisão",
		VehicleID:      uuid.New(),
		OpenedByUserID: uuid.New(),
	}

	out, err := svc.Create(ctx, input)
	assert.Error(t, err)
	assert.Nil(t, out)
}
