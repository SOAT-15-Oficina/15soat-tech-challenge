package routes

import (
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/config"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/handler"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/routes/middlewares"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
	"github.com/ESSantana/15soat-tech-challenge-step-1/packages/email"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(app *fiber.App, db *pgxpool.Pool, cfg *config.Config, emailProv email.Provider) {
	app.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("Pong")
	})

	registerSwagger(app)
	registerAuth(app, db, cfg.JWT.SecretKey)
	registerCustomer(app, db, cfg.JWT.SecretKey)
	registerVehicle(app, db, cfg.JWT.SecretKey)
	registerSupply(app, db, cfg.JWT.SecretKey)
	registerWorkOrderServicePublic(app, db)
	registerPublicWorkOrder(app, db)
	registerWorkOrder(app, db, cfg.JWT.SecretKey, emailProv, cfg.Server.BaseURL)
	registerWorkshopService(app, db, cfg.JWT.SecretKey)
}

func registerAuth(app *fiber.App, db *pgxpool.Pool, jwtSecretKey string) {
	userRepo := repository.NewUserRepository(db)
	userSvc := service.NewUserService(userRepo, jwtSecretKey)
	authHandler := handler.NewAuthHandler(userSvc)
	userHandler := handler.NewUserHandler(userSvc)

	app.Post("/auth/register", authHandler.Register)
	app.Post("/auth/login", authHandler.Login)

	users := app.Group("/users", middlewares.Auth(jwtSecretKey), middlewares.RequireRoles(middlewares.RoleAdmin))
	users.Get("/", userHandler.GetAll)
	users.Get("/:id", userHandler.GetByID)
	users.Put("/:id", userHandler.Update)
	users.Delete("/:id", userHandler.Delete)
}

func registerCustomer(app *fiber.App, db *pgxpool.Pool, jwtSecretKey string) {
	customerRepo := repository.NewCustomerRepository(db)
	customerSvc := service.NewCustomerService(customerRepo)
	customerHandler := handler.NewCustomerHandler(customerSvc)

	group := app.Group("/customers", middlewares.Auth(jwtSecretKey), middlewares.RequireRoles(middlewares.RoleAdmin, middlewares.RoleEmployee))
	group.Post("/", customerHandler.Create)
	group.Get("/", customerHandler.GetAll)
	group.Get("/:id", customerHandler.GetByID)
	group.Put("/:id", customerHandler.Update)
	group.Delete("/:id", customerHandler.Delete)

}

func registerVehicle(app *fiber.App, db *pgxpool.Pool, jwtSecretKey string) {
	vehicleRepo := repository.NewVehicleRepository(db)
	vehicleSvc := service.NewVehicleService(vehicleRepo)
	vehicleHandler := handler.NewVehicleHandler(vehicleSvc)

	group := app.Group("/vehicles", middlewares.Auth(jwtSecretKey), middlewares.RequireRoles(middlewares.RoleAdmin, middlewares.RoleEmployee))
	group.Post("/", vehicleHandler.Create)
	group.Get("/", vehicleHandler.GetAll)
	group.Get("/:id", vehicleHandler.GetByID)
	group.Put("/:id", vehicleHandler.Update)
	group.Delete("/:id", vehicleHandler.Delete)
}

func registerWorkshopService(app *fiber.App, db *pgxpool.Pool, jwtSecretKey string) {
	wsRepo := repository.NewWorkshopServiceRepository(db)
	wsSvc := service.NewWorkshopServiceService(wsRepo)
	wsHandler := handler.NewWorkshopServiceHandler(wsSvc)

	group := app.Group("/services", middlewares.Auth(jwtSecretKey), middlewares.RequireRoles(middlewares.RoleAdmin, middlewares.RoleEmployee))
	group.Post("/", wsHandler.Create)
	group.Get("/avg-execution-time", wsHandler.GetAvgExecutionTime)
	group.Get("/", wsHandler.GetAll)
	group.Get("/:id", wsHandler.GetByID)
	group.Put("/:id", wsHandler.Update)
	group.Delete("/:id", wsHandler.Delete)
}

func registerSupply(app *fiber.App, db *pgxpool.Pool, jwtSecretKey string) {
	supplyRepo := repository.NewSupplyRepository(db)
	supplySvc := service.NewSupplyService(supplyRepo)
	supplyHandler := handler.NewSupplyHandler(supplySvc)

	group := app.Group("/supplies", middlewares.Auth(jwtSecretKey), middlewares.RequireRoles(middlewares.RoleAdmin, middlewares.RoleEmployee))
	group.Post("/", supplyHandler.Create)
	group.Get("/", supplyHandler.GetAll)
	group.Get("/:id", supplyHandler.GetByID)
	group.Put("/:id", supplyHandler.Update)
	group.Delete("/:id", supplyHandler.Delete)
}

func registerWorkOrder(app *fiber.App, db *pgxpool.Pool, jwtSecretKey string, emailProv email.Provider, baseURL string) {
	workOrderRepo := repository.NewWorkOrderRepository(db)
	wosRepo := repository.NewWorkOrderServiceRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	vehicleRepo := repository.NewVehicleRepository(db)
	wsRepo := repository.NewWorkshopServiceRepository(db)
	supplyRepo := repository.NewSupplyRepository(db)

	statusSvc := service.NewWorkOrderStatusService(workOrderRepo, wosRepo)
	workOrderSvc := service.NewWorkOrderService(workOrderRepo, vehicleRepo)
	budgetSvc := service.NewBudgetService(workOrderRepo, wosRepo, customerRepo, emailProv, baseURL)
	creationSvc := service.NewWorkOrderCreationService(workOrderRepo, wosRepo, wsRepo, supplyRepo, statusSvc)
	workOrderHandler := handler.NewWorkOrderHandler(workOrderSvc, budgetSvc, creationSvc, statusSvc)

	group := app.Group("/work-orders", middlewares.Auth(jwtSecretKey), middlewares.RequireRoles(middlewares.RoleAdmin, middlewares.RoleEmployee))
	group.Post("/", workOrderHandler.Create)
	group.Get("/", workOrderHandler.GetAll)
	group.Get("/:id", workOrderHandler.GetByID)
	group.Put("/:id", workOrderHandler.Update)
	group.Post("/:id/services", workOrderHandler.AddServices)
	group.Delete("/:id/services/:wosId", workOrderHandler.RemoveService)
	group.Post("/:id/services/:wosId/supplies", workOrderHandler.AddSupplies)
	group.Delete("/:id/services/:wosId/supplies/:supplyId", workOrderHandler.RemoveSupplyFromService)
}

func registerPublicWorkOrder(app *fiber.App, db *pgxpool.Pool) {
	woRepo := repository.NewWorkOrderRepository(db)
	customerRepo := repository.NewCustomerRepository(db)
	wosRepo := repository.NewWorkOrderServiceRepository(db)
	publicSvc := service.NewPublicWorkOrderService(woRepo, customerRepo, wosRepo)
	publicHandler := handler.NewPublicWorkOrderHandler(publicSvc)

	public := app.Group("/public/work-orders")
	public.Get("/:code", publicHandler.GetByCode)
}

func registerWorkOrderServicePublic(app *fiber.App, db *pgxpool.Pool) {
	wosRepo := repository.NewWorkOrderServiceRepository(db)
	woRepo := repository.NewWorkOrderRepository(db)
	statusSvc := service.NewWorkOrderStatusService(woRepo, wosRepo)
	itemSvc := service.NewWorkOrderItemService(wosRepo, woRepo, statusSvc)
	wosHandler := handler.NewWorkOrderServiceHandler(itemSvc)

	approval := app.Group("/public/approvals")
	approval.Get("/services/:workOrderServiceId/approve", wosHandler.Approve)
	approval.Get("/services/:workOrderServiceId/reject", wosHandler.Reject)
	approval.Get("/work-orders/:workOrderId/approve-all", wosHandler.ApproveAll)
	approval.Get("/work-orders/:workOrderId/reject-all", wosHandler.RejectAll)
}
