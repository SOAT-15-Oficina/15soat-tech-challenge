package service

import (
	"context"
	"fmt"
	"log"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/ESSantana/15soat-tech-challenge-step-1/packages/email"
)

type WorkOrderStatusNotifier interface {
	NotifyTransition(ctx context.Context, workOrder *domain.WorkOrder, previousStatus domain.WorkOrderStatus)
}

type workOrderStatusNotifier struct {
	custRepo  repository.CustomerRepository
	emailProv email.Provider
	budgetSvc BudgetService
}

func NewWorkOrderStatusNotifier(
	custRepo repository.CustomerRepository,
	emailProv email.Provider,
	budgetSvc BudgetService,
) WorkOrderStatusNotifier {
	return &workOrderStatusNotifier{
		custRepo:  custRepo,
		emailProv: emailProv,
		budgetSvc: budgetSvc,
	}
}

func (n *workOrderStatusNotifier) NotifyTransition(
	ctx context.Context,
	workOrder *domain.WorkOrder,
	previousStatus domain.WorkOrderStatus,
) {
	if workOrder == nil {
		return
	}

	newStatus := workOrder.Status

	if newStatus == domain.WorkOrderStatusWaitingApproval {
		previous := previousStatus
		if err := n.budgetSvc.GenerateAndSendBudget(ctx, workOrder.ID, &previous); err != nil {
			log.Printf("work order status notification: budget email failed for work order %s: %v", workOrder.ID, err)
		}
		return
	}

	customer, err := n.custRepo.FindByID(ctx, workOrder.CustomerID)
	if err != nil {
		log.Printf("work order status notification: find customer for work order %s: %v", workOrder.ID, err)
		return
	}

	body, err := email.RenderStatusChangeEmail(email.StatusChangeEmailData{
		CustomerName:        customer.Name,
		WorkOrderCode:       workOrder.Code,
		PreviousStatusLabel: domain.WorkOrderStatusLabel(previousStatus),
		NewStatusLabel:      domain.WorkOrderStatusLabel(newStatus),
		Message:             statusChangeMessage(previousStatus, newStatus),
	})
	if err != nil {
		log.Printf("work order status notification: render email for work order %s: %v", workOrder.ID, err)
		return
	}

	msg := email.Message{
		To:      []string{customer.Email},
		Subject: fmt.Sprintf("Atualização da OS %s - %s", workOrder.Code, domain.WorkOrderStatusLabel(newStatus)),
		Body:    body,
		HTML:    true,
	}

	if err := n.emailProv.Send(ctx, msg); err != nil {
		log.Printf("work order status notification: send email for work order %s: %v", workOrder.ID, err)
	}
}

func statusChangeMessage(previousStatus, newStatus domain.WorkOrderStatus) string {
	switch newStatus {
	case domain.WorkOrderStatusInDiagnosis:
		return "Sua ordem de serviço entrou em diagnóstico. Em breve nossa equipe concluirá a avaliação do veículo."
	case domain.WorkOrderStatusApproved:
		return "Seu orçamento foi aprovado. Em seguida iniciaremos a execução dos serviços autorizados."
	case domain.WorkOrderStatusCanceled:
		if previousStatus == domain.WorkOrderStatusWaitingApproval {
			return "Seu orçamento foi recusado e a ordem de serviço foi cancelada."
		}
		return "Sua ordem de serviço foi cancelada."
	case domain.WorkOrderStatusInProgress:
		return "A execução dos serviços da sua ordem de serviço foi iniciada."
	case domain.WorkOrderStatusFinished:
		return "Os serviços da sua ordem de serviço foram finalizados."
	case domain.WorkOrderStatusDelivered:
		return "Sua ordem de serviço foi entregue. Obrigado por confiar em nossa oficina."
	default:
		return "O status da sua ordem de serviço foi atualizado."
	}
}
