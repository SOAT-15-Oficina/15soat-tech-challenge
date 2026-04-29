package handler

import (
	"errors"
	"math"
	"strconv"
	"time"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type WorkshopServiceHandler struct {
	svc service.WorkshopServiceService
}

type workshopServiceRequest struct {
	Title                *string  `json:"title"`
	Description          *string  `json:"description"`
	Price                *float64 `json:"price"`
	EstimatedTimeMinutes *int     `json:"estimatedTimeMinutes"`
	Status               *domain.WorkshopServiceStatus `json:"status"`
	Active               *bool    `json:"active"`
}

type workshopServiceResponse struct {
	ID                   uuid.UUID `json:"id"`
	Title                string    `json:"title"`
	Description          string    `json:"description"`
	Price                float64   `json:"price"`
	EstimatedTimeMinutes int       `json:"estimatedTimeMinutes"`
	Status               domain.WorkshopServiceStatus `json:"status"`
	Active               bool      `json:"active"`
	CreatedAt            string    `json:"createdAt"`
	UpdatedAt            string    `json:"updatedAt"`
}

type workshopServiceListResponse struct {
	Items []workshopServiceResponse `json:"items"`
	Page  int                       `json:"page"`
	Limit int                       `json:"limit"`
	Total int                       `json:"total"`
}

type avgExecutionTimeResponse struct {
	ServiceID            uuid.UUID `json:"serviceId"`
	Title                string    `json:"title"`
	EstimatedTimeMinutes int       `json:"estimatedTimeMinutes"`
	AvgRealTimeMinutes   float64   `json:"avgRealTimeMinutes"`
	ExecutionCount       int       `json:"executionCount"`
	DifferenceMinutes    float64   `json:"differenceMinutes"`
}

func NewWorkshopServiceHandler(svc service.WorkshopServiceService) *WorkshopServiceHandler {
	return &WorkshopServiceHandler{svc: svc}
}

func (h *WorkshopServiceHandler) RegisterRoutes(app *fiber.App) {
	group := app.Group("/services")
	group.Post("/", h.Create)
	group.Get("/avg-execution-time", h.GetAvgExecutionTime)
	group.Get("/", h.GetAll)
	group.Get("/:id", h.GetByID)
	group.Put("/:id", h.Update)
	group.Delete("/:id", h.Delete)
}

func (h *WorkshopServiceHandler) Create(c fiber.Ctx) error {
	var req workshopServiceRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	ws, err := req.toCreateDomain()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	result, err := h.svc.Create(c.Context(), ws)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(toResponse(result))
}

func (h *WorkshopServiceHandler) GetAll(c fiber.Ctx) error {
	filters, err := parseListFilters(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	items, total, err := h.svc.List(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	responseItems := make([]workshopServiceResponse, 0, len(items))
	for _, item := range items {
		responseItems = append(responseItems, toResponse(&item))
	}

	return c.JSON(workshopServiceListResponse{
		Items: responseItems,
		Page:  filters.Page,
		Limit: filters.Limit,
		Total: total,
	})
}

func (h *WorkshopServiceHandler) GetByID(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	ws, err := h.svc.GetByID(c.Context(), id)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(toResponse(ws))
}

func (h *WorkshopServiceHandler) Update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req workshopServiceRequest
	if err := c.Bind().JSON(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	input, err := req.toUpdateInput()
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	result, err := h.svc.Update(c.Context(), id, input)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	return c.JSON(toResponse(result))
}

func (h *WorkshopServiceHandler) Delete(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	result, err := h.svc.Delete(c.Context(), id)
	if err != nil {
		return h.handleServiceError(c, err)
	}

	if result.Deactivated {
		return c.JSON(fiber.Map{
			"message": "service has existing work order links and was deactivated",
			"service": toResponse(result.DeactivatedResource),
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

func (h *WorkshopServiceHandler) GetAvgExecutionTime(c fiber.Ctx) error {
	filters, err := parseAvgExecutionTimeFilters(c)
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	results, err := h.svc.GetAvgExecutionTime(c.Context(), filters)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	items := make([]avgExecutionTimeResponse, 0, len(results))
	for _, r := range results {
		items = append(items, avgExecutionTimeResponse{
			ServiceID:            r.ServiceID,
			Title:                r.Title,
			EstimatedTimeMinutes: r.EstimatedTimeMinutes,
			AvgRealTimeMinutes:   math.Round(r.AvgRealTimeMinutes*100) / 100,
			ExecutionCount:       r.ExecutionCount,
			DifferenceMinutes:    math.Round((r.AvgRealTimeMinutes-float64(r.EstimatedTimeMinutes))*100) / 100,
		})
	}

	return c.JSON(items)
}

func parseAvgExecutionTimeFilters(c fiber.Ctx) (domain.AvgExecutionTimeFilters, error) {
	var filters domain.AvgExecutionTimeFilters

	if v := c.Query("from"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return filters, errors.New("from must be in format YYYY-MM-DD")
		}
		filters.From = &t
	}

	if v := c.Query("to"); v != "" {
		t, err := time.Parse("2006-01-02", v)
		if err != nil {
			return filters, errors.New("to must be in format YYYY-MM-DD")
		}
		endOfDay := t.Add(24*time.Hour - time.Nanosecond)
		filters.To = &endOfDay
	}

	if v := c.Query("technicianId"); v != "" {
		id, err := uuid.Parse(v)
		if err != nil {
			return filters, errors.New("technicianId must be a valid UUID")
		}
		filters.TechnicianID = &id
	}

	return filters, nil
}

func (h *WorkshopServiceHandler) handleServiceError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "service not found"})
	case errors.Is(err, service.ErrWorkshopServiceTitleAlreadyExists):
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{"error": err.Error()})
	case errors.Is(err, domain.ErrWorkshopServiceTitleRequired),
		errors.Is(err, domain.ErrWorkshopServiceTitleLength),
		errors.Is(err, domain.ErrWorkshopServiceDescriptionLength),
		errors.Is(err, domain.ErrWorkshopServicePriceMustBePositive),
		errors.Is(err, domain.ErrWorkshopServiceDurationMustBePositive),
		errors.Is(err, domain.ErrWorkshopServiceInvalidStatus):
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
}

func parseListFilters(c fiber.Ctx) (domain.WorkshopServiceListFilters, error) {
	page := 1
	limit := 10

	if v := c.Query("page"); v != "" {
		val, err := strconv.Atoi(v)
		if err != nil || val <= 0 {
			return domain.WorkshopServiceListFilters{}, errors.New("page must be a positive integer")
		}
		page = val
	}

	if v := c.Query("limit"); v != "" {
		val, err := strconv.Atoi(v)
		if err != nil || val <= 0 {
			return domain.WorkshopServiceListFilters{}, errors.New("limit must be a positive integer")
		}
		limit = val
	}

	var active *bool
	if v := c.Query("active"); v != "" {
		val, err := strconv.ParseBool(v)
		if err != nil {
			return domain.WorkshopServiceListFilters{}, errors.New("active must be true or false")
		}
		active = &val
	}

	title := c.Query("title")

	return domain.WorkshopServiceListFilters{
		Active: active,
		Title:  title,
		Page:   page,
		Limit:  limit,
	}, nil
}

func (r workshopServiceRequest) toCreateDomain() (*domain.WorkshopService, error) {
	if r.Title == nil || r.Price == nil || r.EstimatedTimeMinutes == nil {
		return nil, errors.New("title, price and estimatedTimeMinutes are required")
	}

	return &domain.WorkshopService{
		Title:                *r.Title,
		Description:          ptrString(r.Description),
		PriceCents:           priceToCents(*r.Price),
		EstimatedTimeMinutes: *r.EstimatedTimeMinutes,
	}, nil
}

func (r workshopServiceRequest) toUpdateInput() (service.WorkshopServiceUpdateInput, error) {
	input := service.WorkshopServiceUpdateInput{}

	if r.Title != nil {
		input.Title = r.Title
	}
	if r.Description != nil {
		input.Description = r.Description
	}
	if r.Price != nil {
		cents := priceToCents(*r.Price)
		input.PriceCents = &cents
	}
	if r.EstimatedTimeMinutes != nil {
		input.EstimatedTimeMinutes = r.EstimatedTimeMinutes
	}
	if r.Status != nil {
		input.Status = r.Status
	}
	if r.Active != nil {
		input.Active = r.Active
	}

	if input.Title == nil && input.Description == nil && input.PriceCents == nil && input.EstimatedTimeMinutes == nil && input.Status == nil && input.Active == nil {
		return service.WorkshopServiceUpdateInput{}, errors.New("at least one field must be provided")
	}

	return input, nil
}

func toResponse(item *domain.WorkshopService) workshopServiceResponse {
	return workshopServiceResponse{
		ID:                   item.ID,
		Title:                item.Title,
		Description:          item.Description,
		Price:                centsToPrice(item.PriceCents),
		EstimatedTimeMinutes: item.EstimatedTimeMinutes,
		Status:               item.Status,
		Active:               item.Active,
		CreatedAt:            item.CreatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:            item.UpdatedAt.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func ptrString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

func priceToCents(v float64) int {
	return int(math.Round(v * 100))
}

func centsToPrice(v int) float64 {
	return float64(v) / 100
}
