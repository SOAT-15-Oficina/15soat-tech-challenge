package application

import (
    "time"

    "github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
    "github.com/google/uuid"
)

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
    Data       []domain.WorkOrder `json:"data"`
    Total      int                `json:"total"`
    Page       int                `json:"page"`
    Limit      int                `json:"limit"`
    TotalPages int                `json:"total_pages"`
}
