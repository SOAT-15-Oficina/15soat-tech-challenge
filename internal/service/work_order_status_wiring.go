package service

import (
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/application/port"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
)

func NewWorkOrderStatusServiceWithNotifications(
	woRepo repository.WorkOrderRepository,
	wosRepo repository.WorkOrderServiceRepository,
	customerRepo repository.CustomerRepository,
	emailPort port.EmailSender,
	baseURL string,
) WorkOrderStatusService {
	budgetSvc := NewBudgetService(woRepo, wosRepo, customerRepo, emailPort, baseURL)
	notifier := NewWorkOrderStatusNotifier(customerRepo, emailPort, budgetSvc)
	return NewWorkOrderStatusService(woRepo, notifier)
}
