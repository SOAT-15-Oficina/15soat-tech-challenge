package domain

import (
	"time"

	"github.com/google/uuid"
)

type WorkOrderServiceStatusHistory struct {
	ID                 uuid.UUID              `json:"id"`
	WorkOrderServiceID uuid.UUID              `json:"work_order_service_id"`
	Status             WorkOrderServiceStatus `json:"status"`
	ChangedByUserID    *uuid.UUID             `json:"changed_by_user_id,omitempty"`
	Note               *string                `json:"note,omitempty"`
	ChangedAt          time.Time              `json:"changed_at"`
}
