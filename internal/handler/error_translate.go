package handler

import (
    "errors"

    "github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
    "github.com/gofiber/fiber/v3"
)

// mapErrorResponse centralizes mapping of domain/service errors to HTTP responses.
// Returns (true, err) when it already wrote a response to the client.
func mapErrorResponse(c fiber.Ctx, err error, notFoundMsg string) (bool, error) {
    if errors.Is(err, service.ErrWorkOrderInvalidStatusForItems) || errors.Is(err, service.ErrInvalidStatusTransition) {
        return true, c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
    }

    if errors.Is(err, service.ErrWorkshopServiceInactive) {
        return true, c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
    }

    if errors.Is(err, service.ErrWorkOrderServiceOwnership) {
        return true, c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
    }

    if errors.Is(err, service.ErrVehicleNotBelongingToCustomer) {
        return true, c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{"error": err.Error()})
    }

    // Fallback to DB-specific translator which may inspect pgx errors.
    return dbErrResponse(c, err, notFoundMsg)
}
