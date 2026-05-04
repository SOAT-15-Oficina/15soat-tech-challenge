package domain

import (
	"time"

	"github.com/google/uuid"
)

type Vehicle struct {
	ID        		uuid.UUID `json:"id"`
	LicensePlate	string	  `json:"licensePlate"`
	CustomerID		uuid.UUID `json:"customerId"`
	Model			string	  `json:"model"`
	Year 			int		  `json:"year"`
	Brand			string 	  `json:"brand"`
	CreatedAt		time.Time `json:"created_at"`
	UpdatedAt		time.Time `json:"updated_at"`
}