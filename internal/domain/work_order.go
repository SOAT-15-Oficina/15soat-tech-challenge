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
	ID                       uuid.UUID       `json:"id"`
	Code                     string          `json:"code"`
	Title                    string          `json:"title"`
	Description              *string         `json:"description,omitempty"`
	CustomerID               uuid.UUID       `json:"customer_id"`
	VehicleID                uuid.UUID       `json:"vehicle_id"`
	OpenedByUserID           uuid.UUID       `json:"opened_by_user_id"`
	AssignedTechnicianID     *uuid.UUID      `json:"assigned_technician_id,omitempty"`
	Status                   WorkOrderStatus `json:"status"`
	TotalEstimatedPriceCents int             `json:"total_estimated_price_cents"`
	ReceivedAt               time.Time       `json:"received_at"`
	QuoteSentAt              *time.Time      `json:"quote_sent_at,omitempty"`
	ApprovedAt               *time.Time      `json:"approved_at,omitempty"`
	StartedAt                *time.Time      `json:"started_at,omitempty"`
	FinishedAt               *time.Time      `json:"finished_at,omitempty"`
	DeliveredAt              *time.Time      `json:"delivered_at,omitempty"`
	CreatedAt                time.Time       `json:"created_at"`
	UpdatedAt                time.Time       `json:"updated_at"`
}
