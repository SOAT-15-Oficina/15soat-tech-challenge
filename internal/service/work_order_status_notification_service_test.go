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

type mockBudgetService struct {
	mock.Mock
}

func (m *mockBudgetService) GenerateAndSendBudget(ctx context.Context, workOrderID uuid.UUID, previousStatus *domain.WorkOrderStatus) error {
	args := m.Called(ctx, workOrderID, previousStatus)
	return args.Error(0)
}

func TestWorkOrderStatusNotifier_SendsStatusEmail(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	emailProv := new(mockEmailProvider)
	budgetSvc := new(mockBudgetService)
	notifier := NewWorkOrderStatusNotifier(custRepo, emailProv, budgetSvc)
	ctx := context.Background()

	custID := uuid.New()
	wo := &domain.WorkOrder{
		ID:         uuid.New(),
		Code:       "WO-100",
		CustomerID: custID,
		Status:     domain.WorkOrderStatusInProgress,
	}
	customer := &domain.Customer{ID: custID, Name: "Ana", Email: "ana@example.com"}

	custRepo.On("FindByID", ctx, custID).Return(customer, nil)
	emailProv.On("Send", ctx, mock.AnythingOfType("email.Message")).Return(nil)

	notifier.NotifyTransition(ctx, wo, domain.WorkOrderStatusApproved)

	emailProv.AssertExpectations(t)
	msg := emailProv.Calls[0].Arguments.Get(1).(email.Message)
	assert.Equal(t, []string{"ana@example.com"}, msg.To)
	assert.Contains(t, msg.Subject, "WO-100")
	assert.Contains(t, msg.Body, "WO-100")
	assert.Contains(t, msg.Body, "Aprovada")
	assert.Contains(t, msg.Body, "Em execução")
}

func TestWorkOrderStatusNotifier_WaitingApprovalUsesBudgetService(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	emailProv := new(mockEmailProvider)
	budgetSvc := new(mockBudgetService)
	notifier := NewWorkOrderStatusNotifier(custRepo, emailProv, budgetSvc)
	ctx := context.Background()

	woID := uuid.New()
	wo := &domain.WorkOrder{
		ID:     woID,
		Status: domain.WorkOrderStatusWaitingApproval,
	}

	budgetSvc.On("GenerateAndSendBudget", ctx, woID, mock.AnythingOfType("*domain.WorkOrderStatus")).Return(nil)

	notifier.NotifyTransition(ctx, wo, domain.WorkOrderStatusInDiagnosis)

	budgetSvc.AssertExpectations(t)
	emailProv.AssertNotCalled(t, "Send")
}

func TestWorkOrderStatusNotifier_CanceledMessage(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	emailProv := new(mockEmailProvider)
	budgetSvc := new(mockBudgetService)
	notifier := NewWorkOrderStatusNotifier(custRepo, emailProv, budgetSvc)
	ctx := context.Background()

	custID := uuid.New()
	wo := &domain.WorkOrder{
		ID:         uuid.New(),
		Code:       "WO-200",
		CustomerID: custID,
		Status:     domain.WorkOrderStatusCanceled,
	}
	customer := &domain.Customer{ID: custID, Name: "Pedro", Email: "pedro@example.com"}

	custRepo.On("FindByID", ctx, custID).Return(customer, nil)
	emailProv.On("Send", ctx, mock.AnythingOfType("email.Message")).Return(nil)

	notifier.NotifyTransition(ctx, wo, domain.WorkOrderStatusWaitingApproval)

	msg := emailProv.Calls[0].Arguments.Get(1).(email.Message)
	assert.True(t, strings.Contains(msg.Body, "recusado"))
}

func TestWorkOrderStatusNotifier_EmailFailureIsBestEffort(t *testing.T) {
	custRepo := new(mockCustomerRepo)
	emailProv := new(mockEmailProvider)
	budgetSvc := new(mockBudgetService)
	notifier := NewWorkOrderStatusNotifier(custRepo, emailProv, budgetSvc)
	ctx := context.Background()

	custID := uuid.New()
	wo := &domain.WorkOrder{
		ID:         uuid.New(),
		Code:       "WO-300",
		CustomerID: custID,
		Status:     domain.WorkOrderStatusDelivered,
	}
	customer := &domain.Customer{ID: custID, Name: "Luiza", Email: "luiza@example.com"}

	custRepo.On("FindByID", ctx, custID).Return(customer, nil)
	emailProv.On("Send", ctx, mock.AnythingOfType("email.Message")).Return(errors.New("smtp down"))

	require.NotPanics(t, func() {
		notifier.NotifyTransition(ctx, wo, domain.WorkOrderStatusFinished)
	})
}
