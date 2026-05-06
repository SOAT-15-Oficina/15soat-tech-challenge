package handler

import (
	"errors"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/domain"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
	"github.com/gofiber/fiber/v3"
)

type RegisterRequest struct {
	Username string          `json:"username"`
	Password string          `json:"password"`
	Role     domain.UserRole `json:"role"`
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type AuthHandler struct {
	svc service.UserService
}

func NewAuthHandler(svc service.UserService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(c fiber.Ctx) error {
	var body RegisterRequest
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	user, err := h.svc.Register(c.Context(), body.Username, body.Password, body.Role)
	if err != nil {
		if handled, resp := dbErrResponse(c, err, "user not found"); handled {
			return resp
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(user)
}

func (h *AuthHandler) Login(c fiber.Ctx) error {
	var body LoginRequest
	if err := c.Bind().JSON(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	token, err := h.svc.Login(c.Context(), body.Username, body.Password)
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid credentials"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{"token": token})
}
