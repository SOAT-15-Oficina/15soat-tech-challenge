package domain

import (
	"time"

	"github.com/google/uuid"
)

type WorkOrderStatus string

const (
	WorkOrderStatusReceived        WorkOrderStatus = "RECEBIDA"
	WorkOrderStatusInDiagnosis     WorkOrderStatus = "EM_DIAGNOSTICO"
	WorkOrderStatusWaitingApproval WorkOrderStatus = "AGUARDANDO_APROVACAO"
	WorkOrderStatusApproved        WorkOrderStatus = "APROVADO"
	WorkOrderStatusInProgress      WorkOrderStatus = "EM_EXECUCAO"
	WorkOrderStatusFinished        WorkOrderStatus = "FINALIZADA"
	WorkOrderStatusDelivered       WorkOrderStatus = "ENTREGUE"
	WorkOrderStatusCanceled        WorkOrderStatus = "CANCELADA"
)

type WorkOrder struct {
	ID                       uuid.UUID                      `json:"id"`
	Code                     string                         `json:"code"`
	Title                    string                         `json:"title"`
	Description              *string                        `json:"description,omitempty"`
	CustomerID               uuid.UUID                      `json:"-"`
	VehicleID                uuid.UUID                      `json:"-"`
	OpenedByUserID           uuid.UUID                      `json:"opened_by_user_id"`
	AssignedTechnicianID     *uuid.UUID                     `json:"assigned_technician_id,omitempty"`
	Status                   WorkOrderStatus                `json:"status"`
	TotalEstimatedPriceCents int                            `json:"total_estimated_price_cents"`
	ReceivedAt               time.Time                      `json:"received_at"`
	QuoteSentAt              *time.Time                     `json:"quote_sent_at,omitempty"`
	ApprovedAt               *time.Time                     `json:"approved_at,omitempty"`
	StartedAt                *time.Time                     `json:"started_at,omitempty"`
	FinishedAt               *time.Time                     `json:"finished_at,omitempty"`
	DeliveredAt              *time.Time                     `json:"delivered_at,omitempty"`
	CreatedAt                time.Time                      `json:"created_at"`
	UpdatedAt                time.Time                      `json:"updated_at"`
	Customer                 *WorkOrderCustomer             `json:"customer"`
	Vehicle                  *WorkOrderVehicle              `json:"vehicle"`
	Services                 []WorkOrderServiceWithSupplies `json:"services,omitempty"`
}

type WorkOrderCustomer struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Document string    `json:"document"`
}

type WorkOrderVehicle struct {
	ID           uuid.UUID `json:"id"`
	LicensePlate string    `json:"license_plate"`
	Brand        string    `json:"brand"`
	Model        string    `json:"model"`
	Year         int       `json:"year"`
}

type WorkOrderServiceWithSupplies struct {
	ID                        uuid.UUID                        `json:"id"`
	Description               string                           `json:"description"`
	ServicePriceCentsSnapshot int                              `json:"service_price_cents_snapshot"`
	Status                    string                           `json:"status"`
	ApprovalStatus            string                           `json:"approval_status"`
	Quantity                  int                              `json:"quantity"`
	Supplies                  []WorkOrderServiceSupplyResponse `json:"supplies"`
}

type WorkOrderServiceSupplyResponse struct {
	ID                       uuid.UUID `json:"id"`
	Description              string    `json:"description"`
	SupplyPriceCentsSnapshot int       `json:"supply_price_cents_snapshot"`
	SupplyQuantity           int       `json:"supply_quantity"`
}

var WorkOrderListingExcludedStatuses = []WorkOrderStatus{
	WorkOrderStatusFinished,
	WorkOrderStatusDelivered,
	WorkOrderStatusCanceled,
}

var workOrderStatusSortPriority = map[WorkOrderStatus]int{
	WorkOrderStatusInProgress:      1,
	WorkOrderStatusApproved:        2,
	WorkOrderStatusWaitingApproval: 3,
	WorkOrderStatusInDiagnosis:     4,
	WorkOrderStatusReceived:        5,
	WorkOrderStatusCanceled:        6,
}

func WorkOrderStatusSortPriorityOf(status WorkOrderStatus) int {
	if p, ok := workOrderStatusSortPriority[status]; ok {
		return p
	}
	return 99
}

func IsExcludedFromListing(status WorkOrderStatus) bool {
	for _, s := range WorkOrderListingExcludedStatuses {
		if s == status {
			return true
		}
	}
	return false
}

type WorkOrderListFilters struct {
	Status     string    `query:"status"`
	CustomerID uuid.UUID `query:"customerId"`
	VehicleID  uuid.UUID `query:"vehicleId"`
	FromDate   *time.Time
	ToDate     *time.Time
	Page       int
	Limit      int
}

type WorkOrderListResponse struct {
	Data       []WorkOrder `json:"data"`
	Total      int         `json:"total"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalPages int         `json:"total_pages"`
}
