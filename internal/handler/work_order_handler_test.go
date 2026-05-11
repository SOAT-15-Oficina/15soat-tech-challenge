package handler

import (
	"context"
	"net/http"
	"testing"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
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
