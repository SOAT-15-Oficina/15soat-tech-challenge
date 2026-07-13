package service

import (
	"context"
	"fmt"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/application/port"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/ESSantana/15soat-tech-challenge-step-1/packages/email"
	"github.com/google/uuid"
)

type WorkOrderItemService interface {
	ApproveService(ctx context.Context, workOrderServiceID uuid.UUID) error
	RejectService(ctx context.Context, workOrderServiceID uuid.UUID) error
	ApproveAllByWorkOrder(ctx context.Context, workOrderID uuid.UUID) error
	RejectAllByWorkOrder(ctx context.Context, workOrderID uuid.UUID) error
}

type workOrderItemService struct {
	wosRepo   repository.WorkOrderServiceRepository
	woRepo    repository.WorkOrderRepository
	statusSvc WorkOrderStatusService
	emailPort port.EmailSender
	emailTo   string
}

func NewWorkOrderItemService(
	wosRepo repository.WorkOrderServiceRepository,
	woRepo repository.WorkOrderRepository,
	statusSvc WorkOrderStatusService,
	opts ...WorkOrderItemServiceOption,
) WorkOrderItemService {
	svc := &workOrderItemService{
		wosRepo:   wosRepo,
		woRepo:    woRepo,
		statusSvc: statusSvc,
	}
	for _, opt := range opts {
		opt(svc)
	}
	return svc
}

type WorkOrderItemServiceOption func(*workOrderItemService)

func WithPurchaseAlert(sender port.EmailSender, to string) WorkOrderItemServiceOption {
	return func(s *workOrderItemService) {
		s.emailPort = sender
		s.emailTo = to
	}
}

func (s *workOrderItemService) ApproveService(ctx context.Context, workOrderServiceID uuid.UUID) error {
	wos, err := s.wosRepo.FindByID(ctx, workOrderServiceID)
	if err != nil {
		return fmt.Errorf("approve: find service: %w", err)
	}

	if wos.ApprovalStatus != domain.WorkOrderServiceApprovalPending {
		return nil // idempotent
	}

	if err := s.wosRepo.UpdateApprovalStatus(ctx, workOrderServiceID, domain.WorkOrderServiceApprovalApproved); err != nil {
		return fmt.Errorf("approve: update status: %w", err)
	}

	return s.evaluateWorkOrderCompletion(ctx, wos.WorkOrderID)
}

func (s *workOrderItemService) RejectService(ctx context.Context, workOrderServiceID uuid.UUID) error {
	wos, err := s.wosRepo.FindByID(ctx, workOrderServiceID)
	if err != nil {
		return fmt.Errorf("reject: find service: %w", err)
	}

	if wos.ApprovalStatus != domain.WorkOrderServiceApprovalPending {
		return nil // idempotent
	}

	if err := s.wosRepo.UpdateApprovalStatus(ctx, workOrderServiceID, domain.WorkOrderServiceApprovalRejected); err != nil {
		return fmt.Errorf("reject: update status: %w", err)
	}

	return s.evaluateWorkOrderCompletion(ctx, wos.WorkOrderID)
}

func (s *workOrderItemService) ApproveAllByWorkOrder(ctx context.Context, workOrderID uuid.UUID) error {
	if err := s.wosRepo.UpdateApprovalStatusByWorkOrderID(ctx, workOrderID, domain.WorkOrderServiceApprovalApproved); err != nil {
		return fmt.Errorf("approve all: update status: %w", err)
	}

	return s.evaluateWorkOrderCompletion(ctx, workOrderID)
}

func (s *workOrderItemService) RejectAllByWorkOrder(ctx context.Context, workOrderID uuid.UUID) error {
	if err := s.wosRepo.UpdateApprovalStatusByWorkOrderID(ctx, workOrderID, domain.WorkOrderServiceApprovalRejected); err != nil {
		return fmt.Errorf("reject all: update status: %w", err)
	}

	return s.evaluateWorkOrderCompletion(ctx, workOrderID)
}

func (s *workOrderItemService) evaluateWorkOrderCompletion(ctx context.Context, workOrderID uuid.UUID) error {
	services, err := s.wosRepo.FindByWorkOrderID(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("evaluate: find services: %w", err)
	}

	hasApproved := false
	for _, svc := range services {
		if svc.ApprovalStatus == domain.WorkOrderServiceApprovalPending {
			return nil // still pending decisions
		}
		if svc.ApprovalStatus == domain.WorkOrderServiceApprovalApproved {
			hasApproved = true
		}
	}

	var newStatus domain.WorkOrderStatus
	if hasApproved {
		newStatus = domain.WorkOrderStatusApproved
	} else {
		newStatus = domain.WorkOrderStatusCanceled
	}

	wo, err := s.statusSvc.TransitionTo(ctx, workOrderID, newStatus)
	if err != nil {
		return fmt.Errorf("evaluate: transition status: %w", err)
	}

	approvedTotal, err := s.wosRepo.CalculateApprovedTotalForWorkOrder(ctx, workOrderID)
	if err != nil {
		return fmt.Errorf("evaluate: calculate approved total: %w", err)
	}
	wo.TotalEstimatedPriceCents = approvedTotal

	if _, err := s.woRepo.Update(ctx, wo); err != nil {
		return fmt.Errorf("evaluate: update work order: %w", err)
	}

	if hasApproved && s.emailPort != nil {
		s.sendPurchaseAlertIfNeeded(ctx, workOrderID)
	}

	return nil
}

func (s *workOrderItemService) sendPurchaseAlertIfNeeded(ctx context.Context, workOrderID uuid.UUID) {
	shortages, err := s.wosRepo.FindSupplyShortagesByWorkOrderID(ctx, workOrderID)
	if err != nil || len(shortages) == 0 {
		return
	}

	alerts, err := s.wosRepo.FindApprovedServicesWithShortages(ctx)
	if err != nil || len(alerts) == 0 {
		return
	}

	wo, err := s.woRepo.FindByID(ctx, workOrderID)
	if err != nil {
		return
	}

	var items []email.PurchaseAlertItem
	for _, a := range alerts {
		items = append(items, email.PurchaseAlertItem{
			ServiceTitle: a.ServiceTitle,
			SupplyTitle:  a.SupplyTitle,
			Required:     a.Required,
			InStock:      a.InStock,
			ToBuy:        a.Required - a.InStock,
		})
	}

	body, err := email.RenderPurchaseAlertEmail(email.PurchaseAlertEmailData{
		WorkOrderCode:  wo.Code,
		WorkOrderTitle: wo.Title,
		Items:          items,
	})
	if err != nil {
		return
	}

	msg := port.EmailMessage{
		To:      []string{s.emailTo},
		Subject: fmt.Sprintf("Alerta de Compra - OS %s", wo.Code),
		Body:    body,
		HTML:    true,
	}
	_ = s.emailPort.Send(ctx, msg)
}
