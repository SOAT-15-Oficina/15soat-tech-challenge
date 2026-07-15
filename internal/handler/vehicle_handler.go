package handler

import (
	"errors"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type vehicleRequest struct {
	LicensePlate string    `json:"licensePlate"`
	CustomerID   uuid.UUID `json:"customerId"`
	Model        string    `json:"model"`
	Year         int       `json:"year"`
	Brand        string    `json:"brand"`
}

type VehicleHandler struct {
	svc service.VehicleService
}

func NewVehicleHandler(svc service.VehicleService) *VehicleHandler {
	return &VehicleHandler{svc: svc}
}

func (h *VehicleHandler) Create(c fiber.Ctx) error {
	var req vehicleRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	vehicle := domain.Vehicle{
		LicensePlate: req.LicensePlate,
		CustomerID:   req.CustomerID,
		Model:        req.Model,
		Year:         req.Year,
		Brand:        req.Brand,
	}

	result, err := h.svc.Create(c.Context(), &vehicle)
	if err != nil {
		var valErr *domain.VehicleValidationError
		if errors.As(err, &valErr) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if handled, resp := dbErrResponse(c, err, "vehicle not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.Status(fiber.StatusCreated).JSON(toVehicleResponse(result))
}

func (h *VehicleHandler) GetAll(c fiber.Ctx) error {
	filters := domain.VehicleListFilters{}

	customerID, err := queryWithAlias(c, "customer_id", "customerId")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	if customerID != "" {
		id, err := uuid.Parse(customerID)
		if err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "customer_id must be a valid UUID"})
		}
		filters.CustomerID = id
	}

	vehicles, err := h.svc.GetAllWithFilters(c.Context(), filters)
	if err != nil {
		return internalServerError(c)
	}

	if vehicles == nil {
		vehicles = []domain.Vehicle{}
	}

	respItems := make([]vehicleResponse, 0, len(vehicles))
	for i := range vehicles {
		respItems = append(respItems, toVehicleResponse(&vehicles[i]))
	}
	return c.JSON(fiber.Map{"data": respItems})
}

func (h *VehicleHandler) GetByID(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	vehicle, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		if handled, resp := dbErrResponse(c, err, "vehicle not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.JSON(toVehicleResponse(vehicle))
}

func (h *VehicleHandler) Update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req vehicleRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	vehicle := domain.Vehicle{
		ID:           id,
		LicensePlate: req.LicensePlate,
		CustomerID:   req.CustomerID,
		Model:        req.Model,
		Year:         req.Year,
		Brand:        req.Brand,
	}

	result, err := h.svc.Update(c.Context(), &vehicle)
	if err != nil {
		var valErr *domain.VehicleValidationError
		if errors.As(err, &valErr) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if handled, resp := dbErrResponse(c, err, "vehicle not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.JSON(toVehicleResponse(result))
}

func (h *VehicleHandler) Delete(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.svc.Delete(c.Context(), id); err != nil {
		if handled, resp := dbErrResponse(c, err, "vehicle not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.SendStatus(fiber.StatusNoContent)
}
