package routes

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/config"
	"github.com/gofiber/fiber/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterRoutes(t *testing.T) {
	app := fiber.New()
	cfg := &config.Config{
		Server: &config.ServerConfig{BaseURL: "http://localhost:3000"},
		JWT:    &config.JWTConfig{SecretKey: "test-secret"},
	}
	// Should not panic with nil db and nil email provider
	RegisterRoutes(app, nil, cfg, nil)

	// Verify ping route works
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestSwagger(t *testing.T) {
	app := fiber.New()
	registerSwagger(app)

	req := httptest.NewRequest(http.MethodGet, "/swagger", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusOK, resp.StatusCode)
}

func TestSwaggerYAML_NotFound(t *testing.T) {
	app := fiber.New()
	registerSwagger(app)

	req := httptest.NewRequest(http.MethodGet, "/docs/swagger.yaml", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, fiber.StatusNotFound, resp.StatusCode)
}
