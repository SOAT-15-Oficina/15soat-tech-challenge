package handler

import (
	"errors"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type WorkOrderServiceHandler struct {
	svc service.WorkOrderItemService
}

func NewWorkOrderServiceHandler(svc service.WorkOrderItemService) *WorkOrderServiceHandler {
	return &WorkOrderServiceHandler{svc: svc}
}

func (h *WorkOrderServiceHandler) Approve(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("workOrderServiceId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order service id"})
	}

	if err := h.svc.ApproveService(c.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "work order service not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Serviço aprovado com sucesso"})
}

func (h *WorkOrderServiceHandler) Reject(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("workOrderServiceId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order service id"})
	}

	if err := h.svc.RejectService(c.Context(), id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "work order service not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Serviço reprovado com sucesso"})
}

func (h *WorkOrderServiceHandler) ApproveAll(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("workOrderId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order id"})
	}

	if err := h.svc.ApproveAllByWorkOrder(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Todos os serviços foram aprovados com sucesso"})
}

func (h *WorkOrderServiceHandler) RejectAll(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("workOrderId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order id"})
	}

	if err := h.svc.RejectAllByWorkOrder(c.Context(), id); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"message": "Todos os serviços foram reprovados com sucesso"})
}
