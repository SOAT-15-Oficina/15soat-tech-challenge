package handler

import (
	"time"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/google/uuid"
)

type workOrderResponse struct {
	ID                       uuid.UUID                  `json:"id"`
	Code                     string                     `json:"code"`
	Title                    string                     `json:"title"`
	Description              *string                    `json:"description,omitempty"`
	OpenedByUserID           uuid.UUID                  `json:"openedByUserId"`
	AssignedTechnicianID     *uuid.UUID                 `json:"assignedTechnicianId,omitempty"`
	Status                   domain.WorkOrderStatus     `json:"status"`
	TotalEstimatedPriceCents int                        `json:"totalEstimatedPriceCents"`
	ReceivedAt               time.Time                  `json:"receivedAt"`
	QuoteSentAt              *time.Time                 `json:"quoteSentAt,omitempty"`
	ApprovedAt               *time.Time                 `json:"approvedAt,omitempty"`
	StartedAt                *time.Time                 `json:"startedAt,omitempty"`
	FinishedAt               *time.Time                 `json:"finishedAt,omitempty"`
	DeliveredAt              *time.Time                 `json:"deliveredAt,omitempty"`
	CreatedAt                time.Time                  `json:"createdAt"`
	UpdatedAt                time.Time                  `json:"updatedAt"`
	Customer                 *workOrderCustomerResponse `json:"customer"`
	Vehicle                  *workOrderVehicleResponse  `json:"vehicle"`
	AssignedTechnician       *workOrderTechnicianResp   `json:"assignedTechnician"`
	Services                 []workOrderServiceResponse `json:"services"`
}

type workOrderCustomerResponse struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Document string    `json:"document"`
}

type workOrderVehicleResponse struct {
	ID           uuid.UUID `json:"id"`
	LicensePlate string    `json:"licensePlate"`
	Brand        string    `json:"brand"`
	Model        string    `json:"model"`
	Year         int       `json:"year"`
}

type workOrderTechnicianResp struct {
	ID       uuid.UUID     `json:"id"`
	Username string        `json:"username"`
	Role     domain.UserRole `json:"role"`
}

type workOrderServiceResponse struct {
	ID                                  uuid.UUID                              `json:"id"`
	WorkOrderID                         uuid.UUID                              `json:"workOrderId"`
	ServiceID                           uuid.UUID                              `json:"serviceId"`
	ServiceTitleSnapshot                string                                 `json:"serviceTitleSnapshot"`
	ServiceDescriptionSnapshot          *string                                `json:"serviceDescriptionSnapshot,omitempty"`
	ServicePriceCentsSnapshot           int                                    `json:"servicePriceCentsSnapshot"`
	ServiceEstimatedTimeMinutesSnapshot int                                    `json:"serviceEstimatedTimeMinutesSnapshot"`
	ApprovalStatus                      domain.WorkOrderServiceApprovalStatus  `json:"approvalStatus"`
	Status                              domain.WorkOrderServiceStatus          `json:"status"`
	Supplies                            []workOrderServiceSupplyResponse       `json:"supplies"`
	StartedAt                           *time.Time                             `json:"startedAt,omitempty"`
	FinishedAt                          *time.Time                             `json:"finishedAt,omitempty"`
	CreatedAt                           time.Time                              `json:"createdAt"`
	UpdatedAt                           time.Time                              `json:"updatedAt"`
}

type workOrderServiceSupplyResponse struct {
	ID                       uuid.UUID `json:"id"`
	WorkOrderServiceID       uuid.UUID `json:"workOrderServiceId"`
	SupplyID                 uuid.UUID `json:"supplyId"`
	SupplyTitleSnapshot      string    `json:"supplyTitleSnapshot"`
	SupplyPriceCentsSnapshot int       `json:"supplyPriceCentsSnapshot"`
	SupplyQuantity           int       `json:"supplyQuantity"`
	CreatedAt                time.Time `json:"createdAt"`
	UpdatedAt                time.Time `json:"updatedAt"`
}

type workOrderListResponse struct {
	Data       []workOrderResponse `json:"data"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	TotalPages int                 `json:"totalPages"`
}

func toWorkOrderResponse(wo *domain.WorkOrder) workOrderResponse {
	resp := workOrderResponse{
		ID:                       wo.ID,
		Code:                     wo.Code,
		Title:                    wo.Title,
		Description:              wo.Description,
		OpenedByUserID:           wo.OpenedByUserID,
		AssignedTechnicianID:     wo.AssignedTechnicianID,
		Status:                   wo.Status,
		TotalEstimatedPriceCents: wo.TotalEstimatedPriceCents,
		ReceivedAt:               wo.ReceivedAt,
		QuoteSentAt:              wo.QuoteSentAt,
		ApprovedAt:               wo.ApprovedAt,
		StartedAt:                wo.StartedAt,
		FinishedAt:               wo.FinishedAt,
		DeliveredAt:              wo.DeliveredAt,
		CreatedAt:                wo.CreatedAt,
		UpdatedAt:                wo.UpdatedAt,
		Services:                 []workOrderServiceResponse{},
	}
	if wo.Customer != nil {
		resp.Customer = &workOrderCustomerResponse{
			ID:       wo.Customer.ID,
			Name:     wo.Customer.Name,
			Document: wo.Customer.Document,
		}
	}
	if wo.Vehicle != nil {
		resp.Vehicle = &workOrderVehicleResponse{
			ID:           wo.Vehicle.ID,
			LicensePlate: wo.Vehicle.LicensePlate,
			Brand:        wo.Vehicle.Brand,
			Model:        wo.Vehicle.Model,
			Year:         wo.Vehicle.Year,
		}
	}
	if wo.AssignedTechnician != nil {
		resp.AssignedTechnician = &workOrderTechnicianResp{
			ID:       wo.AssignedTechnician.ID,
			Username: wo.AssignedTechnician.Username,
			Role:     wo.AssignedTechnician.Role,
		}
	}
	for _, svc := range wo.Services {
		resp.Services = append(resp.Services, toWorkOrderServiceResponse(&svc))
	}
	return resp
}

func toWorkOrderServiceResponse(svc *domain.WorkOrderService) workOrderServiceResponse {
	resp := workOrderServiceResponse{
		ID:                                  svc.ID,
		WorkOrderID:                         svc.WorkOrderID,
		ServiceID:                           svc.ServiceID,
		ServiceTitleSnapshot:                svc.ServiceTitleSnapshot,
		ServiceDescriptionSnapshot:          svc.ServiceDescriptionSnapshot,
		ServicePriceCentsSnapshot:           svc.ServicePriceCentsSnapshot,
		ServiceEstimatedTimeMinutesSnapshot: svc.ServiceEstimatedTimeMinutesSnapshot,
		ApprovalStatus:                      svc.ApprovalStatus,
		Status:                              svc.Status,
		Supplies:                            []workOrderServiceSupplyResponse{},
		StartedAt:                           svc.StartedAt,
		FinishedAt:                          svc.FinishedAt,
		CreatedAt:                           svc.CreatedAt,
		UpdatedAt:                           svc.UpdatedAt,
	}
	for _, sup := range svc.Supplies {
		resp.Supplies = append(resp.Supplies, workOrderServiceSupplyResponse{
			ID:                       sup.ID,
			WorkOrderServiceID:       sup.WorkOrderServiceID,
			SupplyID:                 sup.SupplyID,
			SupplyTitleSnapshot:      sup.SupplyTitleSnapshot,
			SupplyPriceCentsSnapshot: sup.SupplyPriceCentsSnapshot,
			SupplyQuantity:           sup.SupplyQuantity,
			CreatedAt:                sup.CreatedAt,
			UpdatedAt:                sup.UpdatedAt,
		})
	}
	return resp
}

type customerResponse struct {
	ID           uuid.UUID                 `json:"id"`
	Name         string                    `json:"name"`
	Email        string                    `json:"email"`
	Document     string                    `json:"document"`
	DocumentType domain.CustomerDocumentType `json:"documentType"`
	CreatedAt    time.Time                 `json:"createdAt"`
	UpdatedAt    time.Time                 `json:"updatedAt"`
}

func toCustomerResponse(c *domain.Customer) customerResponse {
	return customerResponse{
		ID:           c.ID,
		Name:         c.Name,
		Email:        c.Email,
		Document:     c.Document,
		DocumentType: c.DocumentType,
		CreatedAt:    c.CreatedAt,
		UpdatedAt:    c.UpdatedAt,
	}
}

type vehicleResponse struct {
	ID           uuid.UUID `json:"id"`
	LicensePlate string    `json:"licensePlate"`
	CustomerID   uuid.UUID `json:"customerId"`
	Model        string    `json:"model"`
	Year         int       `json:"year"`
	Brand        string    `json:"brand"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func toVehicleResponse(v *domain.Vehicle) vehicleResponse {
	return vehicleResponse{
		ID:           v.ID,
		LicensePlate: v.LicensePlate,
		CustomerID:   v.CustomerID,
		Model:        v.Model,
		Year:         v.Year,
		Brand:        v.Brand,
		CreatedAt:    v.CreatedAt,
		UpdatedAt:    v.UpdatedAt,
	}
}

type supplyResponse struct {
	ID            uuid.UUID        `json:"id"`
	Title         string           `json:"title"`
	Type          domain.SupplyType `json:"type"`
	PriceCents    int              `json:"priceCents"`
	StockQuantity int              `json:"stockQuantity"`
	MinimumStock  int              `json:"minimumStock"`
	Active        bool             `json:"active"`
	CreatedAt     time.Time        `json:"createdAt"`
	UpdatedAt     time.Time        `json:"updatedAt"`
}

func toSupplyResponse(s *domain.Supply) supplyResponse {
	return supplyResponse{
		ID:            s.ID,
		Title:         s.Title,
		Type:          s.Type,
		PriceCents:    s.PriceCents,
		StockQuantity: s.StockQuantity,
		MinimumStock:  s.MinimumStock,
		Active:        s.Active,
		CreatedAt:     s.CreatedAt,
		UpdatedAt:     s.UpdatedAt,
	}
}

type userResponse struct {
	ID       uuid.UUID     `json:"id"`
	Username string        `json:"username"`
	Role     domain.UserRole `json:"role"`
}

func toUserResponse(u *domain.User) userResponse {
	return userResponse{
		ID:       u.ID,
		Username: u.Username,
		Role:     u.Role,
	}
}
