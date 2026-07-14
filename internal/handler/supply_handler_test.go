package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- mock SupplyService ---

type mockSupplyService struct {
	mock.Mock
}

func (m *mockSupplyService) Create(ctx context.Context, s *domain.Supply) (*domain.Supply, error) {
	args := m.Called(ctx, s)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Supply), args.Error(1)
}

func (m *mockSupplyService) GetByID(ctx context.Context, id uuid.UUID) (*domain.Supply, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Supply), args.Error(1)
}

func (m *mockSupplyService) GetAll(ctx context.Context) ([]domain.Supply, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Supply), args.Error(1)
}

func (m *mockSupplyService) Update(ctx context.Context, s *domain.Supply) (*domain.Supply, error) {
	args := m.Called(ctx, s)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Supply), args.Error(1)
}

func (m *mockSupplyService) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// --- mock WorkOrderServiceRepository (only FindApprovedServicesWithShortages is used) ---

type mockWOSRepo struct {
	mock.Mock
}

func (m *mockWOSRepo) FindApprovedServicesWithShortages(ctx context.Context) ([]repository.SupplyShortageAlert, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.SupplyShortageAlert), args.Error(1)
}

// Stubs for the rest of the interface -- unused by the handler under test.

func (m *mockWOSRepo) Create(ctx context.Context, wos *domain.WorkOrderService) (*domain.WorkOrderService, error) {
	return nil, nil
}
func (m *mockWOSRepo) CreateBatch(ctx context.Context, items []*domain.WorkOrderService) ([]*domain.WorkOrderService, error) {
	return nil, nil
}
func (m *mockWOSRepo) CreateSupply(ctx context.Context, supply *domain.WorkOrderServiceSupply) (*domain.WorkOrderServiceSupply, error) {
	return nil, nil
}
func (m *mockWOSRepo) CreateSupplyBatch(ctx context.Context, items []*domain.WorkOrderServiceSupply) ([]*domain.WorkOrderServiceSupply, error) {
	return nil, nil
}
func (m *mockWOSRepo) DeleteSupplyForWorkOrderService(ctx context.Context, workOrderServiceID, supplyID uuid.UUID) error {
	return nil
}
func (m *mockWOSRepo) DeleteSuppliesByWorkOrderServiceID(ctx context.Context, workOrderServiceID uuid.UUID) error {
	return nil
}
func (m *mockWOSRepo) DeleteByID(ctx context.Context, id uuid.UUID) error { return nil }
func (m *mockWOSRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrderService, error) {
	return nil, nil
}
func (m *mockWOSRepo) FindByWorkOrderID(ctx context.Context, workOrderID uuid.UUID) ([]domain.WorkOrderService, error) {
	return nil, nil
}
func (m *mockWOSRepo) FindSupplyShortagesByWorkOrderID(ctx context.Context, workOrderID uuid.UUID) (map[uuid.UUID]bool, error) {
	return nil, nil
}
func (m *mockWOSRepo) UpdateApprovalStatus(ctx context.Context, id uuid.UUID, status domain.WorkOrderServiceApprovalStatus) error {
	return nil
}
func (m *mockWOSRepo) UpdateApprovalStatusByWorkOrderID(ctx context.Context, workOrderID uuid.UUID, status domain.WorkOrderServiceApprovalStatus) error {
	return nil
}
func (m *mockWOSRepo) CalculateTotalForWorkOrder(ctx context.Context, workOrderID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockWOSRepo) CalculateApprovedTotalForWorkOrder(ctx context.Context, workOrderID uuid.UUID) (int, error) {
	return 0, nil
}
func (m *mockWOSRepo) MarkAsStartedByWorkOrderID(ctx context.Context, workOrderID uuid.UUID, startedAt time.Time) error {
	return nil
}
func (m *mockWOSRepo) MarkAsFinishedByWorkOrderID(ctx context.Context, workOrderID uuid.UUID, finishedAt time.Time) error {
	return nil
}
func (m *mockWOSRepo) MarkServiceAsFinished(ctx context.Context, id uuid.UUID, finishedAt time.Time) error {
	return nil
}
func (m *mockWOSRepo) MarkServiceAsStarted(ctx context.Context, id uuid.UUID, startedAt time.Time) error {
	return nil
}
func (m *mockWOSRepo) HasSupplyShortagesForService(ctx context.Context, workOrderServiceID uuid.UUID) (bool, error) {
	return false, nil
}

// --- helpers ---

func setupSupplyApp(svc *mockSupplyService, wosRepo *mockWOSRepo) *fiber.App {
	app := fiber.New()
	h := NewSupplyHandler(svc, wosRepo)
	app.Post("/supplies", h.Create)
	app.Get("/supplies/pending-purchases", h.PendingPurchases)
	app.Get("/supplies", h.GetAll)
	app.Get("/supplies/:id", h.GetByID)
	app.Put("/supplies/:id", h.Update)
	app.Delete("/supplies/:id", h.Delete)
	return app
}

func supplyJSON(s *domain.Supply) []byte {
	b, _ := json.Marshal(s)
	return b
}

func sampleSupply() *domain.Supply {
	return &domain.Supply{
		ID:            uuid.New(),
		Title:         "Parafuso M6",
		Type:          "material",
		PriceCents:    150,
		StockQuantity: 100,
		MinimumStock:  10,
		Active:        true,
	}
}

// --- tests ---

func TestSupplyCreate_Success(t *testing.T) {
	svc := new(mockSupplyService)
	repo := new(mockWOSRepo)
	app := setupSupplyApp(svc, repo)
	s := sampleSupply()

	svc.On("Create", mock.Anything, mock.AnythingOfType("*domain.Supply")).Return(s, nil)

	req := httptest.NewRequest(http.MethodPost, "/supplies", bytes.NewReader(supplyJSON(s)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestSupplyCreate_InvalidJSON(t *testing.T) {
	svc := new(mockSupplyService)
	repo := new(mockWOSRepo)
	app := setupSupplyApp(svc, repo)

	req := httptest.NewRequest(http.MethodPost, "/supplies", bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestSupplyCreate_ServiceError(t *testing.T) {
	svc := new(mockSupplyService)
	repo := new(mockWOSRepo)
	app := setupSupplyApp(svc, repo)
	s := sampleSupply()

	svc.On("Create", mock.Anything, mock.AnythingOfType("*domain.Supply")).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodPost, "/supplies", bytes.NewReader(supplyJSON(s)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestSupplyGetAll_Success(t *testing.T) {
	svc := new(mockSupplyService)
	repo := new(mockWOSRepo)
	app := setupSupplyApp(svc, repo)

	svc.On("GetAll", mock.Anything).Return([]domain.Supply{*sampleSupply()}, nil)

	req := httptest.NewRequest(http.MethodGet, "/supplies", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestSupplyGetAll_Error(t *testing.T) {
	svc := new(mockSupplyService)
	repo := new(mockWOSRepo)
	app := setupSupplyApp(svc, repo)

	svc.On("GetAll", mock.Anything).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/supplies", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestSupplyGetByID_Success(t *testing.T) {
	svc := new(mockSupplyService)
	repo := new(mockWOSRepo)
	app := setupSupplyApp(svc, repo)
	s := sampleSupply()

	svc.On("GetByID", mock.Anything, s.ID).Return(s, nil)

	req := httptest.NewRequest(http.MethodGet, "/supplies/"+s.ID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestSupplyGetByID_InvalidID(t *testing.T) {
	svc := new(mockSupplyService)
	repo := new(mockWOSRepo)
	app := setupSupplyApp(svc, repo)

	req := httptest.NewRequest(http.MethodGet, "/supplies/not-a-uuid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestSupplyGetByID_NotFound(t *testing.T) {
	svc := new(mockSupplyService)
	repo := new(mockWOSRepo)
	app := setupSupplyApp(svc, repo)
	id := uuid.New()

	svc.On("GetByID", mock.Anything, id).Return(nil, pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/supplies/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}

func TestSupplyUpdate_Success(t *testing.T) {
	svc := new(mockSupplyService)
	repo := new(mockWOSRepo)
	app := setupSupplyApp(svc, repo)
	s := sampleSupply()

	svc.On("Update", mock.Anything, mock.AnythingOfType("*domain.Supply")).Return(s, nil)

	req := httptest.NewRequest(http.MethodPut, "/supplies/"+s.ID.String(), bytes.NewReader(supplyJSON(s)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestSupplyUpdate_InvalidID(t *testing.T) {
	svc := new(mockSupplyService)
	repo := new(mockWOSRepo)
	app := setupSupplyApp(svc, repo)

	req := httptest.NewRequest(http.MethodPut, "/supplies/not-a-uuid", bytes.NewReader(supplyJSON(sampleSupply())))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestSupplyUpdate_InvalidJSON(t *testing.T) {
	svc := new(mockSupplyService)
	repo := new(mockWOSRepo)
	app := setupSupplyApp(svc, repo)
	id := uuid.New()

	req := httptest.NewRequest(http.MethodPut, "/supplies/"+id.String(), bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestSupplyDelete_Success(t *testing.T) {
	svc := new(mockSupplyService)
	repo := new(mockWOSRepo)
	app := setupSupplyApp(svc, repo)
	id := uuid.New()

	svc.On("Delete", mock.Anything, id).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/supplies/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, resp.StatusCode)
}

func TestSupplyDelete_InvalidID(t *testing.T) {
	svc := new(mockSupplyService)
	repo := new(mockWOSRepo)
	app := setupSupplyApp(svc, repo)

	req := httptest.NewRequest(http.MethodDelete, "/supplies/bad-id", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestSupplyDelete_Error(t *testing.T) {
	svc := new(mockSupplyService)
	repo := new(mockWOSRepo)
	app := setupSupplyApp(svc, repo)
	id := uuid.New()

	svc.On("Delete", mock.Anything, id).Return(errors.New("db error"))

	req := httptest.NewRequest(http.MethodDelete, "/supplies/"+id.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}

func TestSupplyPendingPurchases_Success(t *testing.T) {
	svc := new(mockSupplyService)
	repo := new(mockWOSRepo)
	app := setupSupplyApp(svc, repo)

	alerts := []repository.SupplyShortageAlert{
		{
			WorkOrderCode:  "WO-001",
			WorkOrderTitle: "Test WO",
			ServiceTitle:   "Service A",
			SupplyTitle:    "Supply X",
			SupplyID:       uuid.New(),
			Required:       10,
			InStock:        3,
		},
	}
	repo.On("FindApprovedServicesWithShortages", mock.Anything).Return(alerts, nil)

	req := httptest.NewRequest(http.MethodGet, "/supplies/pending-purchases", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	var payload map[string][]map[string]any
	require.NoError(t, json.NewDecoder(resp.Body).Decode(&payload))
	item := payload["data"][0]
	assert.Equal(t, "WO-001", item["work_order_code"])
	assert.Equal(t, "Test WO", item["work_order_title"])
	assert.Equal(t, "Service A", item["service_title"])
	assert.Equal(t, "Supply X", item["supply_title"])
	assert.NotEmpty(t, item["supply_id"])
	assert.Equal(t, float64(10), item["required"])
	assert.Equal(t, float64(3), item["in_stock"])
}

func TestSupplyPendingPurchases_Error(t *testing.T) {
	svc := new(mockSupplyService)
	repo := new(mockWOSRepo)
	app := setupSupplyApp(svc, repo)

	repo.On("FindApprovedServicesWithShortages", mock.Anything).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/supplies/pending-purchases", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
