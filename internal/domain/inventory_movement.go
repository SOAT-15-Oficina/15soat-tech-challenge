package domain

import (
	"time"

	"github.com/google/uuid"
)

type InventoryMovementType string

const (
	InventoryMovementTypeIn         InventoryMovementType = "ENTRADA"
	InventoryMovementTypeOut        InventoryMovementType = "SAIDA"
	InventoryMovementTypeAdjustment InventoryMovementType = "AJUSTE"
)

type InventoryMovement struct {
	ID                 uuid.UUID             `json:"id"`
	SupplyID           uuid.UUID             `json:"supply_id"`
	MovementType       InventoryMovementType `json:"movement_type"`
	Quantity           int                   `json:"quantity"`
	Reason             *string               `json:"reason,omitempty"`
	WorkOrderID        *uuid.UUID            `json:"work_order_id,omitempty"`
	WorkOrderServiceID *uuid.UUID            `json:"work_order_service_id,omitempty"`
	CreatedByUserID    *uuid.UUID            `json:"created_by_user_id,omitempty"`
	CreatedAt          time.Time             `json:"created_at"`
}
