package handler

import (
	"errors"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/auth"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type WorkOrderHandler struct {
	svc         service.WorkOrderService
	budgetSvc   service.BudgetService
	creationSvc service.WorkOrderCreationService
	statusSvc   service.WorkOrderStatusService
	userRepo    repository.UserRepository
}

func NewWorkOrderHandler(svc service.WorkOrderService, budgetSvc service.BudgetService, creationSvc service.WorkOrderCreationService, statusSvc service.WorkOrderStatusService, userRepo repository.UserRepository) *WorkOrderHandler {
	return &WorkOrderHandler{svc: svc, budgetSvc: budgetSvc, creationSvc: creationSvc, statusSvc: statusSvc, userRepo: userRepo}
}

type addServiceRequest struct {
	ServiceID            uuid.UUID `json:"service_id"`
	EstimatedTimeMinutes *int      `json:"estimated_time_minutes,omitempty"`
}

type addSupplyRequest struct {
	SupplyID uuid.UUID `json:"supply_id"`
	Quantity int       `json:"quantity"`
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

	// Handle status transition separately via state machine
	if workOrder.Status != "" {
		var changedByUserID *uuid.UUID
		if claims, ok := c.Locals("token").(*auth.AppClaims); ok {
			user, err := h.userRepo.FindByUsername(c.Context(), claims.User)
			if err == nil {
				changedByUserID = &user.ID
			}
		}

		result, err := h.statusSvc.TransitionTo(c.Context(), id, workOrder.Status, changedByUserID)
		if err != nil {
			if errors.Is(err, service.ErrInvalidStatusTransition) {
				return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
			}
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

		// Clear status so the field update below skips it
		workOrder.Status = ""
	}

	result, err := h.svc.Update(c.Context(), &workOrder)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "work order not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(result)
}

func (h *WorkOrderHandler) AddServices(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order id"})
	}

	var reqs []addServiceRequest
	if err := c.Bind().JSON(&reqs); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	items := make([]service.AddWorkOrderServiceInput, len(reqs))
	for i, r := range reqs {
		items[i] = service.AddWorkOrderServiceInput{
			ServiceID:            r.ServiceID,
			EstimatedTimeMinutes: r.EstimatedTimeMinutes,
		}
	}

	result, err := h.creationSvc.AddServices(c.Context(), id, items)
	if err != nil {
		if errors.Is(err, service.ErrWorkOrderInvalidStatusForItems) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrWorkshopServiceInactive) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "resource not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *WorkOrderHandler) RemoveService(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order id"})
	}

	wosID, err := uuid.Parse(c.Params("wosId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order service id"})
	}

	if err := h.creationSvc.RemoveService(c.Context(), id, wosID); err != nil {
		if errors.Is(err, service.ErrWorkOrderServiceOwnership) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrWorkOrderInvalidStatusForItems) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "resource not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *WorkOrderHandler) AddSupplies(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order id"})
	}

	wosID, err := uuid.Parse(c.Params("wosId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order service id"})
	}

	var reqs []addSupplyRequest
	if err := c.Bind().JSON(&reqs); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	items := make([]service.AddWorkOrderSupplyInput, len(reqs))
	for i, r := range reqs {
		items[i] = service.AddWorkOrderSupplyInput{
			SupplyID: r.SupplyID,
			Quantity: r.Quantity,
		}
	}

	result, err := h.creationSvc.AddSupplies(c.Context(), id, wosID, items)
	if err != nil {
		if errors.Is(err, service.ErrWorkOrderServiceOwnership) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "resource not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *WorkOrderHandler) RemoveSupplyFromService(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order id"})
	}

	wosID, err := uuid.Parse(c.Params("wosId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid work order service id"})
	}

	supplyID, err := uuid.Parse(c.Params("supplyId"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid supply id"})
	}

	if err := h.creationSvc.RemoveSupplyFromService(c.Context(), id, wosID, supplyID); err != nil {
		if errors.Is(err, service.ErrWorkOrderServiceOwnership) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, service.ErrWorkOrderInvalidStatusForItems) {
			return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "resource not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
