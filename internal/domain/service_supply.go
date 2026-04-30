package domain

import (
	"time"

	"github.com/google/uuid"
)

type ServiceSupply struct {
	ID        uuid.UUID `json:"id"`
	ServiceID uuid.UUID `json:"service_id"`
	ItemID    uuid.UUID `json:"item_id"`
	Quantity  int       `json:"quantity"`
	CreatedAt time.Time `json:"created_at"`
}
