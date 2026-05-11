package handler

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// --- mock (reuses mockUserService from user_handler_test.go within the same package) ---
// The mockUserService type is already defined in user_handler_test.go.
// We use a separate type here to avoid redeclaration errors.

type mockAuthUserService struct {
	mock.Mock
}

func (m *mockAuthUserService) Register(ctx context.Context, username, password string, role domain.UserRole) (*domain.User, error) {
	args := m.Called(ctx, username, password, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockAuthUserService) Login(ctx context.Context, username, password string) (string, error) {
	args := m.Called(ctx, username, password)
	return args.String(0), args.Error(1)
}

func (m *mockAuthUserService) GetByID(ctx context.Context, id uuid.UUID) (*domain.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockAuthUserService) GetAll(ctx context.Context) ([]domain.User, error) {
	args := m.Called(ctx)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.User), args.Error(1)
}

func (m *mockAuthUserService) Update(ctx context.Context, id uuid.UUID, username string, role domain.UserRole) (*domain.User, error) {
	args := m.Called(ctx, id, username, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.User), args.Error(1)
}

func (m *mockAuthUserService) Delete(ctx context.Context, id uuid.UUID) error {
	return m.Called(ctx, id).Error(0)
}

// --- helpers ---

func setupAuthApp(svc *mockAuthUserService) *fiber.App {
	app := fiber.New()
	h := NewAuthHandler(svc)
	app.Post("/auth/register", h.Register)
	app.Post("/auth/login", h.Login)
	return app
}

// --- tests ---

func TestRegister_Success(t *testing.T) {
	svc := new(mockAuthUserService)
	app := setupAuthApp(svc)
	u := &domain.User{ID: uuid.New(), Username: "admin", Role: domain.UserRoleAdmin}

	svc.On("Register", mock.Anything, "admin", "secret123", domain.UserRoleAdmin).Return(u, nil)

	body := `{"username":"admin","password":"secret123","role":"admin"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusCreated, resp.StatusCode)
}

func TestRegister_InvalidJSON(t *testing.T) {
	svc := new(mockAuthUserService)
	app := setupAuthApp(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestRegister_Error(t *testing.T) {
	svc := new(mockAuthUserService)
	app := setupAuthApp(svc)

	svc.On("Register", mock.Anything, "admin", "secret123", domain.UserRoleAdmin).Return(nil, errors.New("username already exists"))

	body := `{"username":"admin","password":"secret123","role":"admin"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/register", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestLogin_Success(t *testing.T) {
	svc := new(mockAuthUserService)
	app := setupAuthApp(svc)

	svc.On("Login", mock.Anything, "admin", "secret123").Return("jwt-token", nil)

	body := `{"username":"admin","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestLogin_InvalidJSON(t *testing.T) {
	svc := new(mockAuthUserService)
	app := setupAuthApp(svc)

	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte("{invalid")))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusBadRequest, resp.StatusCode)
}

func TestLogin_InvalidCredentials(t *testing.T) {
	svc := new(mockAuthUserService)
	app := setupAuthApp(svc)

	svc.On("Login", mock.Anything, "admin", "wrong").Return("", service.ErrInvalidCredentials)

	body := `{"username":"admin","password":"wrong"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusUnauthorized, resp.StatusCode)
}

func TestLogin_ServerError(t *testing.T) {
	svc := new(mockAuthUserService)
	app := setupAuthApp(svc)

	svc.On("Login", mock.Anything, "admin", "secret123").Return("", errors.New("db error"))

	body := `{"username":"admin","password":"secret123"}`
	req := httptest.NewRequest(http.MethodPost, "/auth/login", bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusInternalServerError, resp.StatusCode)
}
