package service

import (
	"context"
	"fmt"
	"time"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/ESSantana/15soat-tech-challenge-step-1/packages/email"
	"github.com/google/uuid"
)

type BudgetService interface {
	GenerateAndSendBudget(ctx context.Context, workOrderID uuid.UUID) error
}

type budgetService struct {
	woRepo    repository.WorkOrderRepository
	wosRepo   repository.WorkOrderServiceRepository
	custRepo  repository.CustomerRepository
	emailProv email.Provider
	baseURL   string
}

func NewBudgetService(
	woRepo repository.WorkOrderRepository,
	wosRepo repository.WorkOrderServiceRepository,
	custRepo repository.CustomerRepository,
	emailProv email.Provider,
	baseURL string,
) BudgetService {
	return &budgetService{
		woRepo:    woRepo,
		wosRepo:   wosRepo,
		custRepo:  custRepo,
		emailProv: emailProv,
		baseURL:   baseURL,
	}
}

func (s *budgetService) GenerateAndSendBudget(ctx context.Context, workOrderID uuid.UUID) error {
	services, err := s.wosRepo.FindByWorkOrderID(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("budget: find services: %w", err)
	}

	totalCents, err := s.wosRepo.CalculateTotalForWorkOrder(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("budget: calculate total: %w", err)
	}

	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("budget: find work order: %w", err)
	}

	wo.TotalEstimatedPriceCents = totalCents
	now := time.Now()
	wo.QuoteSentAt = &now
	if _, err := s.woRepo.Update(ctx, wo); err != nil {
		return fmt.Errorf("budget: update work order: %w", err)
	}

	customer, err := s.custRepo.FindByID(ctx, wo.CustomerID)
	if err != nil {
		return fmt.Errorf("budget: find customer: %w", err)
	}

	var serviceItems []email.BudgetServiceItem
	for _, svc := range services {
		serviceItems = append(serviceItems, email.BudgetServiceItem{
			Title:       svc.ServiceTitleSnapshot,
			Amount:      formatCents(svc.ServicePriceCentsSnapshot),
			ApproveLink: fmt.Sprintf("%s/approvals/services/%s/approve", s.baseURL, svc.ID),
			RejectLink:  fmt.Sprintf("%s/approvals/services/%s/reject", s.baseURL, svc.ID),
		})
	}

	data := email.BudgetEmailData{
		CustomerName:   customer.Name,
		Amount:         formatCents(totalCents),
		BudgetLink:     fmt.Sprintf("%s/work-orders/%s", s.baseURL, workOrderID),
		Services:       serviceItems,
		ApproveAllLink: fmt.Sprintf("%s/approvals/work-orders/%s/approve-all", s.baseURL, workOrderID),
		RejectAllLink:  fmt.Sprintf("%s/approvals/work-orders/%s/reject-all", s.baseURL, workOrderID),
	}

	body, err := email.RenderBudgetEmail(data)
	if err != nil {
		return fmt.Errorf("budget: render email: %w", err)
	}

	msg := email.Message{
		To:      []string{customer.Email},
		Subject: fmt.Sprintf("Orçamento - OS %s", wo.Code),
		Body:    body,
		HTML:    true,
	}

	if err := s.emailProv.Send(ctx, msg); err != nil {
		return fmt.Errorf("budget: send email: %w", err)
	}

	return nil
}

func formatCents(cents int) string {
	reais := cents / 100
	centavos := cents % 100
	return fmt.Sprintf("R$ %d,%02d", reais, centavos)
}
