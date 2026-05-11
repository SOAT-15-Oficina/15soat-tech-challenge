package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/auth"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockWorkOrderService struct {
	mock.Mock
}

func (m *mockWorkOrderService) Create(ctx context.Context, wo *domain.WorkOrder) (*domain.WorkOrder, error) {
	args := m.Called(ctx, wo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrder), args.Error(1)
}

func (m *mockWorkOrderService) GetByID(ctx context.Context, id uuid.UUID) (*domain.WorkOrder, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrder), args.Error(1)
}

func (m *mockWorkOrderService) GetAll(ctx context.Context) ([]domain.WorkOrder, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.WorkOrder), args.Error(1)
}

func (m *mockWorkOrderService) GetAllWithFilters(ctx context.Context, filters domain.WorkOrderListFilters) (*domain.WorkOrderListResponse, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrderListResponse), args.Error(1)
}

func (m *mockWorkOrderService) Update(ctx context.Context, wo *domain.WorkOrder) (*domain.WorkOrder, error) {
	args := m.Called(ctx, wo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrder), args.Error(1)
}

func (m *mockWorkOrderService) GetAvgExecutionTime(ctx context.Context, filters domain.AvgExecutionTimeFilters) ([]domain.AvgExecutionTimeResult, error) {
	args := m.Called(ctx, filters)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.AvgExecutionTimeResult), args.Error(1)
}

func setupWorkOrderTestApp(svc service.WorkOrderService) *fiber.App {
	app := fiber.New()
	h := NewWorkOrderHandler(svc, nil, nil, nil, repository.NewUserRepository(nil))
	group := app.Group("/work-orders")
	group.Get("/avg-execution-time", h.GetAvgExecutionTime)
	return app
}

func TestWorkOrder_GetAvgExecutionTime_200(t *testing.T) {
	svcMock := new(mockWorkOrderService)
	app := setupWorkOrderTestApp(svcMock)

	results := []domain.AvgExecutionTimeResult{
		{
			ServiceID:            uuid.New(),
			Title:                "Oil Change",
			EstimatedTimeMinutes: 30,
			AvgRealTimeMinutes:   25.5,
			ExecutionCount:       3,
		},
	}
	svcMock.On("GetAvgExecutionTime", mock.Anything, domain.AvgExecutionTimeFilters{}).Return(results, nil)

	req, _ := http.NewRequest("GET", "/work-orders/avg-execution-time", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	result := parseBody(t, resp)
	items := result["data"].([]any)
	assert.Len(t, items, 1)
	first := items[0].(map[string]any)
	assert.Equal(t, "Oil Change", first["title"])
	assert.Equal(t, float64(25.5), first["avg_real_time_minutes"])
	assert.Equal(t, float64(3), first["execution_count"])
	assert.Equal(t, float64(-4.5), first["difference_minutes"])
}

func TestWorkOrder_GetAvgExecutionTime_200_Empty(t *testing.T) {
	svcMock := new(mockWorkOrderService)
	app := setupWorkOrderTestApp(svcMock)

	svcMock.On("GetAvgExecutionTime", mock.Anything, domain.AvgExecutionTimeFilters{}).Return([]domain.AvgExecutionTimeResult{}, nil)

	req, _ := http.NewRequest("GET", "/work-orders/avg-execution-time", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)

	result := parseBody(t, resp)
	items := result["data"].([]any)
	assert.Len(t, items, 0)
}

func TestWorkOrder_GetAvgExecutionTime_400_InvalidFromDate(t *testing.T) {
	svcMock := new(mockWorkOrderService)
	app := setupWorkOrderTestApp(svcMock)

	req, _ := http.NewRequest("GET", "/work-orders/avg-execution-time?from=bad-date", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestWorkOrder_GetAvgExecutionTime_400_InvalidToDate(t *testing.T) {
	svcMock := new(mockWorkOrderService)
	app := setupWorkOrderTestApp(svcMock)

	req, _ := http.NewRequest("GET", "/work-orders/avg-execution-time?to=bad-date", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestWorkOrder_GetAvgExecutionTime_400_InvalidTechnicianId(t *testing.T) {
	svcMock := new(mockWorkOrderService)
	app := setupWorkOrderTestApp(svcMock)

	req, _ := http.NewRequest("GET", "/work-orders/avg-execution-time?technicianId=not-uuid", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestWorkOrder_GetAvgExecutionTime_WithFilters(t *testing.T) {
	svcMock := new(mockWorkOrderService)
	app := setupWorkOrderTestApp(svcMock)

	svcMock.On("GetAvgExecutionTime", mock.Anything, mock.AnythingOfType("domain.AvgExecutionTimeFilters")).Return([]domain.AvgExecutionTimeResult{}, nil)

	techID := uuid.New()
	req, _ := http.NewRequest("GET", "/work-orders/avg-execution-time?from=2026-01-01&to=2026-12-31&technicianId="+techID.String(), nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

// --- Additional mocks for full work order handler testing ---

type mockBudgetService struct{ mock.Mock }

func (m *mockBudgetService) GenerateAndSendBudget(ctx context.Context, workOrderID uuid.UUID) error {
	return m.Called(ctx, workOrderID).Error(0)
}

type mockCreationService struct{ mock.Mock }

func (m *mockCreationService) AddServices(ctx context.Context, woID uuid.UUID, items []service.AddWorkOrderServiceInput) ([]domain.WorkOrderService, error) {
	args := m.Called(ctx, woID, items)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.WorkOrderService), args.Error(1)
}
func (m *mockCreationService) AddSupplies(ctx context.Context, woID, wosID uuid.UUID, items []service.AddWorkOrderSupplyInput) ([]domain.WorkOrderServiceSupply, error) {
	args := m.Called(ctx, woID, wosID, items)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.WorkOrderServiceSupply), args.Error(1)
}
func (m *mockCreationService) RemoveSupplyFromService(ctx context.Context, woID, wosID, supplyID uuid.UUID) error {
	return m.Called(ctx, woID, wosID, supplyID).Error(0)
}
func (m *mockCreationService) RemoveService(ctx context.Context, woID, wosID uuid.UUID) error {
	return m.Called(ctx, woID, wosID).Error(0)
}
func (m *mockCreationService) StartService(ctx context.Context, woID, wosID uuid.UUID) (bool, error) {
	args := m.Called(ctx, woID, wosID)
	return args.Bool(0), args.Error(1)
}
func (m *mockCreationService) FinalizeService(ctx context.Context, woID, wosID uuid.UUID) error {
	return m.Called(ctx, woID, wosID).Error(0)
}

type mockStatusSvc struct{ mock.Mock }

func (m *mockStatusSvc) TransitionTo(ctx context.Context, workOrderID uuid.UUID, newStatus domain.WorkOrderStatus) (*domain.WorkOrder, error) {
	args := m.Called(ctx, workOrderID, newStatus)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.WorkOrder), args.Error(1)
}
func (m *mockStatusSvc) IsValidTransition(from, to domain.WorkOrderStatus) bool {
	return m.Called(from, to).Bool(0)
}

type mockUserRepo struct{ mock.Mock }

func (m *mockUserRepo) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) FindByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) FindByUsername(ctx context.Context, username string) (*domain.User, error) {
	args := m.Called(ctx, username)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) FindAll(ctx context.Context) ([]domain.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.User), args.Error(1)
}
func (m *mockUserRepo) Update(ctx context.Context, user *domain.User) (*domain.User, error) {
	args := m.Called(ctx, user)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}
func (m *mockUserRepo) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// --- Full setup ---

type woTestDeps struct {
	woSvc       *mockWorkOrderService
	budgetSvc   *mockBudgetService
	creationSvc *mockCreationService
	statusSvc   *mockStatusSvc
	userRepo    *mockUserRepo
}

func setupFullWorkOrderApp() (*fiber.App, *woTestDeps) {
	deps := &woTestDeps{
		woSvc:       new(mockWorkOrderService),
		budgetSvc:   new(mockBudgetService),
		creationSvc: new(mockCreationService),
		statusSvc:   new(mockStatusSvc),
		userRepo:    new(mockUserRepo),
	}
	app := fiber.New()
	h := NewWorkOrderHandler(deps.woSvc, deps.budgetSvc, deps.creationSvc, deps.statusSvc, deps.userRepo)
	g := app.Group("/work-orders")
	g.Get("/", h.GetAll)
	g.Get("/:id", h.GetByID)
	g.Post("/", func(c fiber.Ctx) error {
		c.Locals("token", &auth.AppClaims{User: "testuser", Role: "admin"})
		return c.Next()
	}, h.Create)
	g.Put("/:id", h.Update)
	g.Post("/:id/services", h.AddServices)
	g.Delete("/:id/services/:wosId", h.RemoveService)
	g.Post("/:id/services/:wosId/supplies", h.AddSupplies)
	g.Delete("/:id/services/:wosId/supplies/:supplyId", h.RemoveSupplyFromService)
	g.Post("/:id/services/:wosId/start", h.StartService)
	g.Post("/:id/services/:wosId/finalize", h.FinalizeService)
	return app, deps
}

// --- GetAll tests ---

func TestWorkOrder_GetAll_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	resp := &domain.WorkOrderListResponse{
		Data: []domain.WorkOrder{{ID: uuid.New(), Title: "test"}}, Total: 1, Page: 1, Limit: 10, TotalPages: 1,
	}
	deps.woSvc.On("GetAllWithFilters", mock.Anything, mock.Anything).Return(resp, nil)

	req := httptest.NewRequest(http.MethodGet, "/work-orders/", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_GetAll_WithFilters(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	resp := &domain.WorkOrderListResponse{Data: []domain.WorkOrder{}, Total: 0, Page: 2, Limit: 5, TotalPages: 0}
	deps.woSvc.On("GetAllWithFilters", mock.Anything, mock.Anything).Return(resp, nil)

	custID := uuid.New()
	vehID := uuid.New()
	req := httptest.NewRequest(http.MethodGet, "/work-orders/?page=2&limit=5&status=RECEBIDA&customerId="+custID.String()+"&vehicleId="+vehID.String()+"&from=2026-01-01&to=2026-12-31", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_GetAll_NilResult(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	deps.woSvc.On("GetAllWithFilters", mock.Anything, mock.Anything).Return(nil, nil)

	req := httptest.NewRequest(http.MethodGet, "/work-orders/", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_GetAll_Error(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	deps.woSvc.On("GetAllWithFilters", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	req := httptest.NewRequest(http.MethodGet, "/work-orders/", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, r.StatusCode)
}

// --- GetByID tests ---

func TestWorkOrder_GetByID_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	wo := &domain.WorkOrder{ID: id, Title: "test"}
	deps.woSvc.On("GetByID", mock.Anything, id).Return(wo, nil)

	req := httptest.NewRequest(http.MethodGet, "/work-orders/"+id.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_GetByID_InvalidID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodGet, "/work-orders/bad-id", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_GetByID_NotFound(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	deps.woSvc.On("GetByID", mock.Anything, id).Return(nil, pgx.ErrNoRows)

	req := httptest.NewRequest(http.MethodGet, "/work-orders/"+id.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, r.StatusCode)
}

func TestWorkOrder_GetByID_ServerError(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	deps.woSvc.On("GetByID", mock.Anything, id).Return(nil, errors.New("unexpected"))

	req := httptest.NewRequest(http.MethodGet, "/work-orders/"+id.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, r.StatusCode)
}

// --- Create tests ---

func TestWorkOrder_Create_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	user := &domain.User{ID: uuid.New(), Username: "testuser"}
	deps.userRepo.On("FindByUsername", mock.Anything, "testuser").Return(user, nil)
	wo := &domain.WorkOrder{ID: uuid.New(), Title: "test", Code: "OS-123"}
	deps.woSvc.On("Create", mock.Anything, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	body, _ := json.Marshal(map[string]any{"title": "test", "customer_id": uuid.New(), "vehicle_id": uuid.New()})
	req := httptest.NewRequest(http.MethodPost, "/work-orders/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, r.StatusCode)
}

func TestWorkOrder_Create_NoToken(t *testing.T) {
	deps := &woTestDeps{
		woSvc: new(mockWorkOrderService), budgetSvc: new(mockBudgetService),
		creationSvc: new(mockCreationService), statusSvc: new(mockStatusSvc), userRepo: new(mockUserRepo),
	}
	app := fiber.New()
	h := NewWorkOrderHandler(deps.woSvc, deps.budgetSvc, deps.creationSvc, deps.statusSvc, deps.userRepo)
	app.Post("/work-orders", h.Create) // no middleware to set token

	body, _ := json.Marshal(map[string]any{"title": "test"})
	req := httptest.NewRequest(http.MethodPost, "/work-orders", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, r.StatusCode)
}

func TestWorkOrder_Create_UserRepoError(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	deps.userRepo.On("FindByUsername", mock.Anything, "testuser").Return(nil, errors.New("db error"))

	body, _ := json.Marshal(map[string]any{"title": "test", "customer_id": uuid.New(), "vehicle_id": uuid.New()})
	req := httptest.NewRequest(http.MethodPost, "/work-orders/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, r.StatusCode)
}

func TestWorkOrder_Create_ServiceError(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	user := &domain.User{ID: uuid.New(), Username: "testuser"}
	deps.userRepo.On("FindByUsername", mock.Anything, "testuser").Return(user, nil)
	deps.woSvc.On("Create", mock.Anything, mock.AnythingOfType("*domain.WorkOrder")).Return(nil, errors.New("validation error"))

	body, _ := json.Marshal(map[string]any{"title": "test", "customer_id": uuid.New(), "vehicle_id": uuid.New()})
	req := httptest.NewRequest(http.MethodPost, "/work-orders/", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, r.StatusCode)
}

// --- Update tests ---

func TestWorkOrder_Update_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	wo := &domain.WorkOrder{ID: id, Title: "updated"}
	deps.woSvc.On("Update", mock.Anything, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	body, _ := json.Marshal(map[string]any{"title": "updated"})
	req := httptest.NewRequest(http.MethodPut, "/work-orders/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_Update_InvalidID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()

	body, _ := json.Marshal(map[string]any{"title": "updated"})
	req := httptest.NewRequest(http.MethodPut, "/work-orders/bad-id", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_Update_WithStatusTransition(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	wo := &domain.WorkOrder{ID: id, Status: domain.WorkOrderStatusInDiagnosis}
	deps.statusSvc.On("TransitionTo", mock.Anything, id, domain.WorkOrderStatusInDiagnosis).Return(wo, nil)
	deps.woSvc.On("Update", mock.Anything, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	body, _ := json.Marshal(map[string]any{"status": domain.WorkOrderStatusInDiagnosis})
	req := httptest.NewRequest(http.MethodPut, "/work-orders/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_Update_StatusTransitionInvalid(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	deps.statusSvc.On("TransitionTo", mock.Anything, id, mock.Anything).Return(nil, service.ErrInvalidStatusTransition)

	body, _ := json.Marshal(map[string]any{"status": "ENTREGUE"})
	req := httptest.NewRequest(http.MethodPut, "/work-orders/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

func TestWorkOrder_Update_StatusWaitingApproval_SendsBudget(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	wo := &domain.WorkOrder{ID: id, Status: domain.WorkOrderStatusWaitingApproval}
	deps.statusSvc.On("TransitionTo", mock.Anything, id, domain.WorkOrderStatusWaitingApproval).Return(wo, nil)
	deps.budgetSvc.On("GenerateAndSendBudget", mock.Anything, id).Return(nil)
	deps.woSvc.On("Update", mock.Anything, mock.AnythingOfType("*domain.WorkOrder")).Return(wo, nil)

	body, _ := json.Marshal(map[string]any{"status": domain.WorkOrderStatusWaitingApproval})
	req := httptest.NewRequest(http.MethodPut, "/work-orders/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
	deps.budgetSvc.AssertCalled(t, "GenerateAndSendBudget", mock.Anything, id)
}

func TestWorkOrder_Update_BudgetSendFails(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	wo := &domain.WorkOrder{ID: id, Status: domain.WorkOrderStatusWaitingApproval}
	deps.statusSvc.On("TransitionTo", mock.Anything, id, domain.WorkOrderStatusWaitingApproval).Return(wo, nil)
	deps.budgetSvc.On("GenerateAndSendBudget", mock.Anything, id).Return(errors.New("email fail"))

	body, _ := json.Marshal(map[string]any{"status": domain.WorkOrderStatusWaitingApproval})
	req := httptest.NewRequest(http.MethodPut, "/work-orders/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, r.StatusCode)
}

func TestWorkOrder_Update_NotFound(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	id := uuid.New()
	deps.woSvc.On("Update", mock.Anything, mock.AnythingOfType("*domain.WorkOrder")).Return(nil, pgx.ErrNoRows)

	body, _ := json.Marshal(map[string]any{"title": "x"})
	req := httptest.NewRequest(http.MethodPut, "/work-orders/"+id.String(), bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, r.StatusCode)
}

// --- AddServices tests ---

func TestWorkOrder_AddServices_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	result := []domain.WorkOrderService{{ID: uuid.New(), WorkOrderID: woID}}
	deps.creationSvc.On("AddServices", mock.Anything, woID, mock.Anything).Return(result, nil)
	deps.woSvc.On("GetByID", mock.Anything, woID).Return(&domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusInDiagnosis}, nil)

	body, _ := json.Marshal([]map[string]any{{"service_id": uuid.New()}})
	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, r.StatusCode)
}

func TestWorkOrder_AddServices_InvalidID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodPost, "/work-orders/bad-id/services", bytes.NewReader([]byte("[]")))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_AddServices_InvalidStatus(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	deps.creationSvc.On("AddServices", mock.Anything, woID, mock.Anything).Return(nil, service.ErrWorkOrderInvalidStatusForItems)

	body, _ := json.Marshal([]map[string]any{{"service_id": uuid.New()}})
	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

func TestWorkOrder_AddServices_InactiveService(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	deps.creationSvc.On("AddServices", mock.Anything, woID, mock.Anything).Return(nil, service.ErrWorkshopServiceInactive)

	body, _ := json.Marshal([]map[string]any{{"service_id": uuid.New()}})
	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

// --- RemoveService tests ---

func TestWorkOrder_RemoveService_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("RemoveService", mock.Anything, woID, wosID).Return(nil)

	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+woID.String()+"/services/"+wosID.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, r.StatusCode)
}

func TestWorkOrder_RemoveService_InvalidWoID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodDelete, "/work-orders/bad-id/services/"+uuid.New().String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_RemoveService_InvalidWosID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+uuid.New().String()+"/services/bad-id", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_RemoveService_Ownership(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("RemoveService", mock.Anything, woID, wosID).Return(service.ErrWorkOrderServiceOwnership)

	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+woID.String()+"/services/"+wosID.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

func TestWorkOrder_RemoveService_InvalidStatus(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("RemoveService", mock.Anything, woID, wosID).Return(service.ErrWorkOrderInvalidStatusForItems)

	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+woID.String()+"/services/"+wosID.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

// --- AddSupplies tests ---

func TestWorkOrder_AddSupplies_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	result := []domain.WorkOrderServiceSupply{{ID: uuid.New()}}
	deps.creationSvc.On("AddSupplies", mock.Anything, woID, wosID, mock.Anything).Return(result, nil)
	deps.woSvc.On("GetByID", mock.Anything, woID).Return(&domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusInDiagnosis}, nil)

	body, _ := json.Marshal([]map[string]any{{"supply_id": uuid.New(), "quantity": 2}})
	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/supplies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, r.StatusCode)
}

func TestWorkOrder_AddSupplies_InvalidWoID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodPost, "/work-orders/bad/services/"+uuid.New().String()+"/supplies", bytes.NewReader([]byte("[]")))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_AddSupplies_InvalidWosID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+uuid.New().String()+"/services/bad/supplies", bytes.NewReader([]byte("[]")))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_AddSupplies_Ownership(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("AddSupplies", mock.Anything, woID, wosID, mock.Anything).Return(nil, service.ErrWorkOrderServiceOwnership)

	body, _ := json.Marshal([]map[string]any{{"supply_id": uuid.New(), "quantity": 1}})
	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/supplies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

// --- StartService tests ---

func TestWorkOrder_StartService_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("StartService", mock.Anything, woID, wosID).Return(false, nil)
	deps.woSvc.On("GetByID", mock.Anything, woID).Return(&domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusInProgress}, nil)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/start", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_StartService_WithDelay(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("StartService", mock.Anything, woID, wosID).Return(true, nil)
	deps.woSvc.On("GetByID", mock.Anything, woID).Return(&domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusInProgress}, nil)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/start", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_StartService_InvalidWoID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodPost, "/work-orders/bad/services/"+uuid.New().String()+"/start", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_StartService_InvalidWosID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+uuid.New().String()+"/services/bad/start", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_StartService_NotInProgress(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("StartService", mock.Anything, woID, wosID).Return(false, service.ErrWorkOrderNotInProgress)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/start", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

func TestWorkOrder_StartService_NotApproved(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("StartService", mock.Anything, woID, wosID).Return(false, service.ErrServiceNotApproved)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/start", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

func TestWorkOrder_StartService_NotPending(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("StartService", mock.Anything, woID, wosID).Return(false, service.ErrServiceNotPending)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/start", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

// --- FinalizeService tests ---

func TestWorkOrder_FinalizeService_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("FinalizeService", mock.Anything, woID, wosID).Return(nil)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/finalize", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, r.StatusCode)
}

func TestWorkOrder_FinalizeService_InvalidWoID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodPost, "/work-orders/bad/services/"+uuid.New().String()+"/finalize", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_FinalizeService_InvalidWosID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+uuid.New().String()+"/services/bad/finalize", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_FinalizeService_NotInProgress(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("FinalizeService", mock.Anything, woID, wosID).Return(service.ErrWorkOrderNotInProgress)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/finalize", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

func TestWorkOrder_FinalizeService_ServiceNotInProgress(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	deps.creationSvc.On("FinalizeService", mock.Anything, woID, wosID).Return(service.ErrServiceNotInProgress)

	req := httptest.NewRequest(http.MethodPost, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/finalize", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

// --- RemoveSupplyFromService tests ---

func TestWorkOrder_RemoveSupply_Success(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	supplyID := uuid.New()
	deps.creationSvc.On("RemoveSupplyFromService", mock.Anything, woID, wosID, supplyID).Return(nil)
	deps.woSvc.On("GetByID", mock.Anything, woID).Return(&domain.WorkOrder{ID: woID, Status: domain.WorkOrderStatusInDiagnosis}, nil)

	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/supplies/"+supplyID.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNoContent, r.StatusCode)
}

func TestWorkOrder_RemoveSupply_InvalidWoID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodDelete, "/work-orders/bad/services/"+uuid.New().String()+"/supplies/"+uuid.New().String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_RemoveSupply_InvalidWosID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+uuid.New().String()+"/services/bad/supplies/"+uuid.New().String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_RemoveSupply_InvalidSupplyID(t *testing.T) {
	app, _ := setupFullWorkOrderApp()
	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+uuid.New().String()+"/services/"+uuid.New().String()+"/supplies/bad", nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, r.StatusCode)
}

func TestWorkOrder_RemoveSupply_Ownership(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	supplyID := uuid.New()
	deps.creationSvc.On("RemoveSupplyFromService", mock.Anything, woID, wosID, supplyID).Return(service.ErrWorkOrderServiceOwnership)

	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/supplies/"+supplyID.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

func TestWorkOrder_RemoveSupply_InvalidStatus(t *testing.T) {
	app, deps := setupFullWorkOrderApp()
	woID := uuid.New()
	wosID := uuid.New()
	supplyID := uuid.New()
	deps.creationSvc.On("RemoveSupplyFromService", mock.Anything, woID, wosID, supplyID).Return(service.ErrWorkOrderInvalidStatusForItems)

	req := httptest.NewRequest(http.MethodDelete, "/work-orders/"+woID.String()+"/services/"+wosID.String()+"/supplies/"+supplyID.String(), nil)
	r, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnprocessableEntity, r.StatusCode)
}

// --- GetAvgExecutionTime error path ---

func TestWorkOrder_GetAvgExecutionTime_500(t *testing.T) {
	svcMock := new(mockWorkOrderService)
	app := setupWorkOrderTestApp(svcMock)
	svcMock.On("GetAvgExecutionTime", mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	req, _ := http.NewRequest("GET", "/work-orders/avg-execution-time", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
