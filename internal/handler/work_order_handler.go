package handler

import (
	"errors"
	"strconv"
	"time"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

type WorkOrderHandler struct {
	svc         service.WorkOrderService
	budgetSvc   service.BudgetService
	creationSvc service.WorkOrderCreationService
	statusSvc   service.WorkOrderStatusService
}

func NewWorkOrderHandler(svc service.WorkOrderService, budgetSvc service.BudgetService, creationSvc service.WorkOrderCreationService, statusSvc service.WorkOrderStatusService) *WorkOrderHandler {
	return &WorkOrderHandler{svc: svc, budgetSvc: budgetSvc, creationSvc: creationSvc, statusSvc: statusSvc}
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
		if handled, resp := dbErrResponse(c, err, "work order not found"); handled {
			return resp
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(result)
}

func (h *WorkOrderHandler) GetAll(c fiber.Ctx) error {
	filters := domain.WorkOrderListFilters{
		Page:  1,
		Limit: 10,
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil && p > 0 {
			filters.Page = p
		}
	}
	if limit := c.Query("limit"); limit != "" {
		if l, err := strconv.Atoi(limit); err == nil && l > 0 && l <= 100 {
			filters.Limit = l
		}
	}

	if status := c.Query("status"); status != "" {
		filters.Status = status
	}

	if customerID := c.Query("customerId"); customerID != "" {
		if id, err := uuid.Parse(customerID); err == nil {
			filters.CustomerID = id
		}
	}

	if vehicleID := c.Query("vehicleId"); vehicleID != "" {
		if id, err := uuid.Parse(vehicleID); err == nil {
			filters.VehicleID = id
		}
	}

	if from := c.Query("from"); from != "" {
		if t, err := time.Parse("2006-01-02", from); err == nil {
			filters.FromDate = &t
		}
	}

	if to := c.Query("to"); to != "" {
		if t, err := time.Parse("2006-01-02", to); err == nil {
			t = t.Add(24*time.Hour - 1*time.Nanosecond)
			filters.ToDate = &t
		}
	}

	result, err := h.svc.GetAllWithFilters(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	if result == nil || result.Data == nil {
		result = &domain.WorkOrderListResponse{
			Data:       []domain.WorkOrder{},
			Total:      0,
			Page:       filters.Page,
			Limit:      filters.Limit,
			TotalPages: 0,
		}
	}

	return c.JSON(result)
}

func (h *WorkOrderHandler) GetByID(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	workOrder, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		if handled, resp := dbErrResponse(c, err, "work order not found"); handled {
			return resp
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

	if workOrder.Status != "" {
		result, err := h.statusSvc.TransitionTo(c.Context(), id, workOrder.Status)
		if err != nil {
			if errors.Is(err, service.ErrInvalidStatusTransition) {
				return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
			}
			if handled, resp := dbErrResponse(c, err, "work order not found"); handled {
				return resp
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
		}

		if result.Status == domain.WorkOrderStatusWaitingApproval && h.budgetSvc != nil {
			if err := h.budgetSvc.GenerateAndSendBudget(c.Context(), result.ID); err != nil {
				return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to send budget email: " + err.Error()})
			}
		}

		workOrder.Status = ""
	}

	result, err := h.svc.Update(c.Context(), &workOrder)
	if err != nil {
		if handled, resp := dbErrResponse(c, err, "work order not found"); handled {
			return resp
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
		if handled, resp := dbErrResponse(c, err, "resource not found"); handled {
			return resp
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
		if handled, resp := dbErrResponse(c, err, "resource not found"); handled {
			return resp
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
		if handled, resp := dbErrResponse(c, err, "resource not found"); handled {
			return resp
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
		if handled, resp := dbErrResponse(c, err, "resource not found"); handled {
			return resp
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(fiber.StatusNoContent)
}
