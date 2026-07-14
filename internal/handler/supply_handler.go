package handler

import (
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type supplyRequest struct {
	Title         string           `json:"title"`
	Type          domain.SupplyType `json:"type"`
	PriceCents    int              `json:"priceCents"`
	StockQuantity int              `json:"stockQuantity"`
	MinimumStock  int              `json:"minimumStock"`
	Active        bool             `json:"active"`
}

type SupplyHandler struct {
	svc service.SupplyService
}

func NewSupplyHandler(svc service.SupplyService) *SupplyHandler {
	return &SupplyHandler{svc: svc}
}

func (h *SupplyHandler) Create(c fiber.Ctx) error {
	var req supplyRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	supply := domain.Supply{
		Title:         req.Title,
		Type:          req.Type,
		PriceCents:    req.PriceCents,
		StockQuantity: req.StockQuantity,
		MinimumStock:  req.MinimumStock,
		Active:        req.Active,
	}

	result, err := h.svc.Create(c.Context(), &supply)
	if err != nil {
		if handled, resp := dbErrResponse(c, err, "supply not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.Status(fiber.StatusCreated).JSON(toSupplyResponse(result))
}

func (h *SupplyHandler) GetAll(c fiber.Ctx) error {
	supplies, err := h.svc.GetAll(c.Context())
	if err != nil {
		return internalServerError(c)
	}

	if supplies == nil {
		supplies = []domain.Supply{}
	}

	respItems := make([]supplyResponse, 0, len(supplies))
	for i := range supplies {
		respItems = append(respItems, toSupplyResponse(&supplies[i]))
	}
	return c.JSON(fiber.Map{"data": respItems})
}

func (h *SupplyHandler) GetByID(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	supply, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		if handled, resp := dbErrResponse(c, err, "supply not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.JSON(toSupplyResponse(supply))
}

func (h *SupplyHandler) Update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req supplyRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	supply := domain.Supply{
		ID:            id,
		Title:         req.Title,
		Type:          req.Type,
		PriceCents:    req.PriceCents,
		StockQuantity: req.StockQuantity,
		MinimumStock:  req.MinimumStock,
		Active:        req.Active,
	}

	result, err := h.svc.Update(c.Context(), &supply)
	if err != nil {
		if handled, resp := dbErrResponse(c, err, "supply not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.JSON(toSupplyResponse(result))
}

func (h *SupplyHandler) Delete(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.svc.Delete(c.Context(), id); err != nil {
		if handled, resp := dbErrResponse(c, err, "supply not found"); handled {
			return resp
		}
		return internalServerError(c)
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *SupplyHandler) PendingPurchases(c fiber.Ctx) error {
	alerts, err := h.svc.PendingPurchases(c.Context())
	if err != nil {
		return internalServerError(c)
	}
	return c.JSON(fiber.Map{"data": alerts})
}
