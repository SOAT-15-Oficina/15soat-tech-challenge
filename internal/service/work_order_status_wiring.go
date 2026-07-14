package service

import (
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/application"
	"github.com/ESSantana/15soat-tech-challenge-step-1/packages/email"
)

func NewWorkOrderStatusServiceWithNotifications(
	woRepo application.WorkOrderRepository,
	wosRepo application.WorkOrderServiceRepository,
	customerRepo application.CustomerRepository,
	emailProv email.Provider,
	baseURL string,
) WorkOrderStatusService {
	sender := email.NewWorkOrderNotificationSender(emailProv)
	budgetSvc := NewBudgetService(woRepo, wosRepo, customerRepo, sender, baseURL)
	notifier := NewWorkOrderStatusNotifier(customerRepo, sender, budgetSvc)
	return NewWorkOrderStatusService(woRepo, notifier)
}
