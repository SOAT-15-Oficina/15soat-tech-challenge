package service

import (
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/ESSantana/15soat-tech-challenge-step-1/packages/email"
)

func NewWorkOrderStatusServiceWithNotifications(
	woRepo repository.WorkOrderRepository,
	wosRepo repository.WorkOrderServiceRepository,
	customerRepo repository.CustomerRepository,
	emailProv email.Provider,
	baseURL string,
) WorkOrderStatusService {
	budgetSvc := NewBudgetService(woRepo, wosRepo, customerRepo, emailProv, baseURL)
	notifier := NewWorkOrderStatusNotifier(customerRepo, emailProv, budgetSvc)
	return NewWorkOrderStatusService(woRepo, notifier)
}
