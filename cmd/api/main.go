package main

import (
	"fmt"
	"log"
	"os"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/config"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/database"
	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/routes"
	"github.com/ESSantana/15soat-tech-challenge-step-1/packages/email"
	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	cfg       *config.Config
	db        *pgxpool.Pool
	emailProv email.Provider
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
	database.RunMigrations(db)

	emailProv, err = email.New(cfg.Email.Provider, email.Config{
		Host: cfg.Email.Host,
		Port: cfg.Email.Port,
		From: cfg.Email.From,
	})
	if err != nil {
		shutdownApp(err, "Failed to create email provider")
	}

	log.Println("Dependencies initialized successfully")
}

func main() {
	app := fiber.New(fiber.Config{})
	routes.RegisterRoutes(app, db, cfg, emailProv)

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
