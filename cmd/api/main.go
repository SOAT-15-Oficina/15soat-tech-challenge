package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/config"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/handler"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/infra/database"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/repository"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/service"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	cfg *config.Config
	db  *pgxpool.Pool
)

func init() {
	var err error

	cfg, err = config.Load()
	if err != nil {
		shutdownApp(err, "Failed to load configuration")
	}

	db, err = database.NewConnection(cfg.Database)
	if err != nil {
		shutdownApp(err, "Failed to connect to database")
	}

	if err = database.RunMigrations(db); err != nil {
		shutdownApp(err, "Failed to run migrations or migrations already executed")
	}

	log.Println("Dependencies initialized successfully")
}

func main() {
	app := fiber.New(fiber.Config{})

	app.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("Pong")
	})

	supplyRepo := repository.NewSupplyRepository(db)
	supplySvc := service.NewSupplyService(supplyRepo)
	supplyHandler := handler.NewSupplyHandler(supplySvc)
	supplyHandler.RegisterRoutes(app)

	customerRepo := repository.NewCustomerRepository(db)
	customerSvc := service.NewCustomerService(customerRepo)
	customerHandler := handler.NewCustomerHandler(customerSvc)
	customerHandler.RegisterRoutes(app)

	vehicleRepo := repository.NewVehicleRepository(db)
	vehicleSvc := service.NewVehicleService(vehicleRepo)
	vehicleHandler := handler.NewVehicleHandler(vehicleSvc)
	vehicleHandler.RegisterRoutes(app)

	wsRepo := repository.NewWorkshopServiceRepository(db)
	wsSvc := service.NewWorkshopServiceService(wsRepo)
	wsHandler := handler.NewWorkshopServiceHandler(wsSvc)
	wsHandler.RegisterRoutes(app)

	err := app.Listen(":" + cfg.Server.Port)
	if err != nil {
		shutdownApp(err, "Failed to start server")
	}
}
func shutdownApp(err error, message string) {
	if err != nil {
		fmt.Println(message + " - shutdown with error: " + err.Error())
		os.Exit(1)
	}
}
