package handler

import (
	"errors"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type WorkOrderHandler struct {
	svc       service.WorkOrderService
	budgetSvc service.BudgetService
}

func NewWorkOrderHandler(svc service.WorkOrderService, budgetSvc service.BudgetService) *WorkOrderHandler {
	return &WorkOrderHandler{svc: svc, budgetSvc: budgetSvc}
}

func (h *WorkOrderHandler) Create(c fiber.Ctx) error {
	var workOrder domain.WorkOrder
	if err := c.Bind().JSON(&workOrder); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	result, err := h.svc.Create(c.Context(), &workOrder)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *WorkOrderHandler) GetAll(c fiber.Ctx) error {
	workOrders, err := h.svc.GetAll(c.Context())
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if workOrders == nil {
		return c.JSON([]domain.WorkOrder{})
	}

	return c.JSON(workOrders)
}

func (h *WorkOrderHandler) GetByID(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	workOrder, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "work order not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(workOrder)
}

func (h *WorkOrderHandler) Update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var workOrder domain.WorkOrder
	if err := c.Bind().JSON(&workOrder); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}
	workOrder.ID = id

	result, err := h.svc.Update(c.Context(), &workOrder)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "work order not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if result.Status == domain.WorkOrderStatusWaitingApproval && h.budgetSvc != nil {
		if err := h.budgetSvc.GenerateAndSendBudget(c.Context(), result.ID); err != nil {
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to send budget email: " + err.Error()})
		}
	}

	return c.JSON(result)
}
