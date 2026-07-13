package service

import (
	"context"
	"errors"
	"testing"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/application"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockBudgetNotifier struct {
	mock.Mock
}

func (m *mockBudgetNotifier) SendBudget(ctx context.Context, notification application.BudgetNotification) error {
	return m.Called(ctx, notification).Error(0)
}

func newBudgetService(woRepo *mockWorkOrderRepo, wosRepo *mockWorkOrderServiceRepo, custRepo *mockCustomerRepo, notifier *mockBudgetNotifier) BudgetService {
	return NewBudgetService(woRepo, wosRepo, custRepo, notifier, "http://localhost:3000")
}

func TestGenerateAndSendBudget_Success(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	custRepo := new(mockCustomerRepo)
	notifier := new(mockBudgetNotifier)
	svc := newBudgetService(woRepo, wosRepo, custRepo, notifier)
	ctx := context.Background()

	woID := uuid.New()
	custID := uuid.New()

	svcItem := domain.WorkOrderService{
		ID:                        uuid.New(),
		WorkOrderID:               woID,
		ServiceTitleSnapshot:      "Troca de óleo",
		ServicePriceCentsSnapshot: 5000,
	}
	wo := &domain.WorkOrder{
		ID:         woID,
		Code:       "WO-001",
		CustomerID: custID,
	}
	customer := &domain.Customer{
		ID:    custID,
		Name:  "Maria",
		Email: "maria@example.com",
	}

	wosRepo.On("FindByWorkOrderID", ctx, woID).Return([]domain.WorkOrderService{svcItem}, nil)
	wosRepo.On("FindSupplyShortagesByWorkOrderID", ctx, woID).Return(map[uuid.UUID]bool{}, nil)
	wosRepo.On("CalculateTotalForWorkOrder", ctx, woID).Return(5000, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)
	custRepo.On("FindByID", ctx, custID).Return(customer, nil)
	notifier.On("SendBudget", ctx, mock.AnythingOfType("application.BudgetNotification")).Return(nil)

	err := svc.GenerateAndSendBudget(ctx, woID)
	require.NoError(t, err)
	notifier.AssertExpectations(t)
}

func TestGenerateAndSendBudget_FindServicesFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	custRepo := new(mockCustomerRepo)
	notifier := new(mockBudgetNotifier)
	svc := newBudgetService(woRepo, wosRepo, custRepo, notifier)
	ctx := context.Background()
	woID := uuid.New()

	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(nil, errors.New("db error"))

	err := svc.GenerateAndSendBudget(ctx, woID)
	assert.Error(t, err)
	notifier.AssertNotCalled(t, "SendBudget")
}

func TestGenerateAndSendBudget_NotificationFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	custRepo := new(mockCustomerRepo)
	notifier := new(mockBudgetNotifier)
	svc := newBudgetService(woRepo, wosRepo, custRepo, notifier)
	ctx := context.Background()

	woID := uuid.New()
	custID := uuid.New()

	wo := &domain.WorkOrder{ID: woID, Code: "WO-001", CustomerID: custID}
	customer := &domain.Customer{ID: custID, Name: "Maria", Email: "maria@example.com"}

	wosRepo.On("FindByWorkOrderID", ctx, woID).Return([]domain.WorkOrderService{}, nil)
	wosRepo.On("FindSupplyShortagesByWorkOrderID", ctx, woID).Return(map[uuid.UUID]bool{}, nil)
	wosRepo.On("CalculateTotalForWorkOrder", ctx, woID).Return(0, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)
	custRepo.On("FindByID", ctx, custID).Return(customer, nil)
	notifier.On("SendBudget", ctx, mock.AnythingOfType("application.BudgetNotification")).Return(errors.New("smtp error"))

	err := svc.GenerateAndSendBudget(ctx, woID)
	assert.Error(t, err)
}

func TestGenerateAndSendBudget_AddsTwoDaysWhenSupplyIsShort(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	custRepo := new(mockCustomerRepo)
	notifier := new(mockBudgetNotifier)
	svc := newBudgetService(woRepo, wosRepo, custRepo, notifier)
	ctx := context.Background()

	woID := uuid.New()
	custID := uuid.New()
	serviceID := uuid.New()

	svcItem := domain.WorkOrderService{
		ID:                                  serviceID,
		WorkOrderID:                         woID,
		ServiceTitleSnapshot:                "Troca de filtro",
		ServicePriceCentsSnapshot:           7000,
		ServiceEstimatedTimeMinutesSnapshot: 60,
	}
	wo := &domain.WorkOrder{ID: woID, Code: "WO-002", CustomerID: custID}
	customer := &domain.Customer{ID: custID, Name: "Maria", Email: "maria@example.com"}

	wosRepo.On("FindByWorkOrderID", ctx, woID).Return([]domain.WorkOrderService{svcItem}, nil)
	wosRepo.On("FindSupplyShortagesByWorkOrderID", ctx, woID).Return(map[uuid.UUID]bool{serviceID: true}, nil)
	wosRepo.On("CalculateTotalForWorkOrder", ctx, woID).Return(7000, nil)
	woRepo.On("FindByID", ctx, woID).Return(wo, nil)
	woRepo.On("Update", ctx, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)
	custRepo.On("FindByID", ctx, custID).Return(customer, nil)
	notifier.On("SendBudget", ctx, mock.AnythingOfType("application.BudgetNotification")).Return(nil)

	err := svc.GenerateAndSendBudget(ctx, woID)
	require.NoError(t, err)

	callArgs := notifier.Calls[0].Arguments
	notification := callArgs.Get(1).(application.BudgetNotification)
	assert.Equal(t, "2 dias e 1 hora", notification.Services[0].Estimated)
}

func TestGenerateAndSendBudget_FindSupplyShortagesFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	custRepo := new(mockCustomerRepo)
	notifier := new(mockBudgetNotifier)
	svc := newBudgetService(woRepo, wosRepo, custRepo, notifier)
	ctx := context.Background()
	woID := uuid.New()

	wosRepo.On("FindByWorkOrderID", ctx, woID).Return([]domain.WorkOrderService{}, nil)
	wosRepo.On("FindSupplyShortagesByWorkOrderID", ctx, woID).Return(nil, errors.New("db error"))

	err := svc.GenerateAndSendBudget(ctx, woID)
	assert.Error(t, err)
	notifier.AssertNotCalled(t, "SendBudget")
}

// --- formatCents ---

func TestFormatCents(t *testing.T) {
	tests := []struct {
		cents    int
		expected string
	}{
		{0, "R$ 0,00"},
		{100, "R$ 1,00"},
		{5050, "R$ 50,50"},
		{10099, "R$ 100,99"},
	}
	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatCents(tt.cents))
		})
	}
}

func TestFormatEstimatedTimeMinutes(t *testing.T) {
	tests := []struct {
		minutes  int
		expected string
	}{
		{0, "0 min"},
		{1, "1 min"},
		{60, "1 hora"},
		{61, "1 hora e 1 min"},
		{1440, "1 dia"},
		{1500, "1 dia e 1 hora"},
		{2940, "2 dias e 1 hora"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			assert.Equal(t, tt.expected, formatEstimatedTimeMinutes(tt.minutes))
		})
	}
}
