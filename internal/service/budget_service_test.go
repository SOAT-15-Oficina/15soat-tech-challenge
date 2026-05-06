package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/packages/email"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// mockEmailProvider mocks email.Provider
type mockEmailProvider struct {
	mock.Mock
}

func (m *mockEmailProvider) Send(ctx context.Context, msg email.Message) error {
	return m.Called(ctx, msg).Error(0)
}

func newBudgetService(woRepo *mockWorkOrderRepo, wosRepo *mockWorkOrderServiceRepo, custRepo *mockCustomerRepo, prov *mockEmailProvider) BudgetService {
	return NewBudgetService(woRepo, wosRepo, custRepo, prov, "http://localhost:3000")
}

func TestGenerateAndSendBudget_Success(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	custRepo := new(mockCustomerRepo)
	emailProv := new(mockEmailProvider)
	svc := newBudgetService(woRepo, wosRepo, custRepo, emailProv)
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
	emailProv.On("Send", ctx, mock.AnythingOfType("email.Message")).Return(nil)

	err := svc.GenerateAndSendBudget(ctx, woID)
	require.NoError(t, err)
	emailProv.AssertExpectations(t)
}

func TestGenerateAndSendBudget_FindServicesFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	custRepo := new(mockCustomerRepo)
	emailProv := new(mockEmailProvider)
	svc := newBudgetService(woRepo, wosRepo, custRepo, emailProv)
	ctx := context.Background()
	woID := uuid.New()

	wosRepo.On("FindByWorkOrderID", ctx, woID).Return(nil, errors.New("db error"))

	err := svc.GenerateAndSendBudget(ctx, woID)
	assert.Error(t, err)
	emailProv.AssertNotCalled(t, "Send")
}

func TestGenerateAndSendBudget_EmailFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	custRepo := new(mockCustomerRepo)
	emailProv := new(mockEmailProvider)
	svc := newBudgetService(woRepo, wosRepo, custRepo, emailProv)
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
	emailProv.On("Send", ctx, mock.AnythingOfType("email.Message")).Return(errors.New("smtp error"))

	err := svc.GenerateAndSendBudget(ctx, woID)
	assert.Error(t, err)
}

func TestGenerateAndSendBudget_AddsTwoDaysWhenSupplyIsShort(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	custRepo := new(mockCustomerRepo)
	emailProv := new(mockEmailProvider)
	svc := newBudgetService(woRepo, wosRepo, custRepo, emailProv)
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
	emailProv.On("Send", ctx, mock.AnythingOfType("email.Message")).Return(nil)

	err := svc.GenerateAndSendBudget(ctx, woID)
	require.NoError(t, err)

	callArgs := emailProv.Calls[0].Arguments
	msg := callArgs.Get(1).(email.Message)
	assert.True(t, strings.Contains(msg.Body, "Prazo estimado: <strong>2 dias e 1 hora</strong>"))
}

func TestGenerateAndSendBudget_FindSupplyShortagesFails(t *testing.T) {
	woRepo := new(mockWorkOrderRepo)
	wosRepo := new(mockWorkOrderServiceRepo)
	custRepo := new(mockCustomerRepo)
	emailProv := new(mockEmailProvider)
	svc := newBudgetService(woRepo, wosRepo, custRepo, emailProv)
	ctx := context.Background()
	woID := uuid.New()

	wosRepo.On("FindByWorkOrderID", ctx, woID).Return([]domain.WorkOrderService{}, nil)
	wosRepo.On("FindSupplyShortagesByWorkOrderID", ctx, woID).Return(nil, errors.New("db error"))

	err := svc.GenerateAndSendBudget(ctx, woID)
	assert.Error(t, err)
	emailProv.AssertNotCalled(t, "Send")
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
