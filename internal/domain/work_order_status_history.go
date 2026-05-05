package domain

import (
	"time"

	"github.com/google/uuid"
)

type WorkOrderStatusHistory struct {
	ID              uuid.UUID       `json:"id"`
	WorkOrderID     uuid.UUID       `json:"work_order_id"`
	FromStatus      WorkOrderStatus `json:"from_status"`
	ToStatus        WorkOrderStatus `json:"to_status"`
	ChangedByUserID *uuid.UUID      `json:"changed_by_user_id,omitempty"`
	Note            *string         `json:"note,omitempty"`
	ChangedAt       time.Time       `json:"changed_at"`
}
