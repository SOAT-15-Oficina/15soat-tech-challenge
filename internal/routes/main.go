package routes

import (
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/config"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/handler"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/routes/middlewares"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RegisterRoutes(app *fiber.App, db *pgxpool.Pool, cfg *config.Config) {
	app.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("Pong")
	})

	registerAuth(app, db, cfg.JWT.SecretKey)
	registerCustomer(app, db, cfg.JWT.SecretKey)
	registerVehicle(app, db, cfg.JWT.SecretKey)
	registerSupply(app, db, cfg.JWT.SecretKey)
	registerWorkOrder(app, db, cfg.JWT.SecretKey)
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

func registerWorkOrder(app *fiber.App, db *pgxpool.Pool, jwtSecretKey string) {
	workOrderRepo := repository.NewWorkOrderRepository(db)
	workOrderSvc := service.NewWorkOrderService(workOrderRepo)
	workOrderHandler := handler.NewWorkOrderHandler(workOrderSvc)

	group := app.Group("/work-orders", middlewares.Auth(jwtSecretKey), middlewares.RequireRoles(middlewares.RoleAdmin, middlewares.RoleEmployee))
	group.Post("/", workOrderHandler.Create)
	group.Get("/", workOrderHandler.GetAll)
	group.Get("/:id", workOrderHandler.GetByID)
	group.Put("/:id", workOrderHandler.Update)
}
