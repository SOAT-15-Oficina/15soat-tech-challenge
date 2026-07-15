package service

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/application"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
)

var ErrWorkOrderNotFound = errors.New("work order not found")

type PublicServiceView struct {
	Title          string                                `json:"title"`
	Status         domain.WorkOrderServiceStatus         `json:"status"`
	ApprovalStatus domain.WorkOrderServiceApprovalStatus `json:"approvalStatus"`
}

type PublicWorkOrderView struct {
	Code                     string                 `json:"code"`
	Status                   domain.WorkOrderStatus `json:"status"`
	TotalEstimatedPriceCents int                    `json:"totalEstimatedPriceCents"`
	ReceivedAt               time.Time              `json:"receivedAt"`
	QuoteSentAt              *time.Time             `json:"quoteSentAt,omitempty"`
	ApprovedAt               *time.Time             `json:"approvedAt,omitempty"`
	StartedAt                *time.Time             `json:"startedAt,omitempty"`
	FinishedAt               *time.Time             `json:"finishedAt,omitempty"`
	DeliveredAt              *time.Time             `json:"deliveredAt,omitempty"`
	Services                 []PublicServiceView    `json:"services"`
}

type PublicWorkOrderService interface {
	GetPublicStatus(ctx context.Context, code, document string) (*PublicWorkOrderView, error)
}

type publicWorkOrderService struct {
	woRepo       application.WorkOrderRepository
	customerRepo application.CustomerRepository
	wosRepo      application.WorkOrderServiceRepository
}

func NewPublicWorkOrderService(
	woRepo application.WorkOrderRepository,
	customerRepo application.CustomerRepository,
	wosRepo application.WorkOrderServiceRepository,
) PublicWorkOrderService {
	return &publicWorkOrderService{
		woRepo:       woRepo,
		customerRepo: customerRepo,
		wosRepo:      wosRepo,
	}
}

var onlyDigits = regexp.MustCompile(`\d+`)

func normalizeDocument(doc string) string {
	return strings.Join(onlyDigits.FindAllString(doc, -1), "")
}

func (s *publicWorkOrderService) GetPublicStatus(ctx context.Context, code, document string) (*PublicWorkOrderView, error) {
	wo, err := s.woRepo.FindByCode(ctx, code)
	if err != nil {
		if errors.Is(err, application.ErrNotFound) {
			return nil, ErrWorkOrderNotFound
		}
		return nil, err
	}

	customer, err := s.customerRepo.FindByID(ctx, wo.CustomerID)
	if err != nil {
		return nil, ErrWorkOrderNotFound
	}

	if normalizeDocument(document) != customer.Document {
		return nil, ErrWorkOrderNotFound
	}

	services, err := s.wosRepo.FindByWorkOrderID(ctx, wo.ID)
	if err != nil {
		return nil, err
	}

	svcViews := make([]PublicServiceView, len(services))
	for i, svc := range services {
		svcViews[i] = PublicServiceView{
			Title:          svc.ServiceTitleSnapshot,
			Status:         svc.Status,
			ApprovalStatus: svc.ApprovalStatus,
		}
	}

	return &PublicWorkOrderView{
		Code:                     wo.Code,
		Status:                   wo.Status,
		TotalEstimatedPriceCents: wo.TotalEstimatedPriceCents,
		ReceivedAt:               wo.ReceivedAt,
		QuoteSentAt:              wo.QuoteSentAt,
		ApprovedAt:               wo.ApprovedAt,
		StartedAt:                wo.StartedAt,
		FinishedAt:               wo.FinishedAt,
		DeliveredAt:              wo.DeliveredAt,
		Services:                 svcViews,
	}, nil
}
