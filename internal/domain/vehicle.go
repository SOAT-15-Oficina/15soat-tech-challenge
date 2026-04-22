package domain

import "github.com/google/uuid"

type Vehicle struct {
	ID        		uuid.UUID `json:"id"`
	LicensePlate	string	  `json:"licensePlate"`
	CustomerID		uuid.UUID `json:"customerId"`
	Model			string	  `json:"model"`
	Year 			int		  `json:"year"`
	Brand			string 	  `json:"brand"`
} 