package handler

import (
	"errors"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type VehicleHandler struct {
	svc service.VehicleService
}

func NewVehicleHandler(svc service.VehicleService) *VehicleHandler {
	return &VehicleHandler{svc: svc}
}

func (h *VehicleHandler) Create(c fiber.Ctx) error {
	var vehicle domain.Vehicle
	if err := c.Bind().JSON(&vehicle); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *VehicleHandler) GetAll(c fiber.Ctx) error {
	filters := domain.VehicleListFilters{}

	if customerID := c.Query("customerId"); customerID != "" {
		if id, err := uuid.Parse(customerID); err == nil {
			filters.CustomerID = id
		}
	}

	vehicles, err := h.svc.GetAllWithFilters(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if vehicles == nil {
		return c.JSON([]domain.Vehicle{})
	}

	return c.JSON(vehicles)
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(vehicle)
}

func (h *VehicleHandler) Update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var vehicle domain.Vehicle
	if err := c.Bind().JSON(&vehicle); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	vehicle.ID = id

	result, err := h.svc.Update(c.Context(), &vehicle)
	if err != nil {
		var valErr *domain.VehicleValidationError
		if errors.As(err, &valErr) {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
		}
		if handled, resp := dbErrResponse(c, err, "vehicle not found"); handled {
			return resp
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
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
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
