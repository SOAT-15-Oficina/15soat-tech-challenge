package main

import (
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/ESSantana/15soat-tech-challenge-step-1/internal/config"
	"github.com/gofiber/fiber/v3"
)

var (
	cfg *config.Config
)

func init() {
	var err error

	fmt.Println(os.Getenv("DATABASE_USER"))
	cfg, err = config.Load()
	if err != nil {
		shutdownApp(err, "Failed to load configuration")
	}
	initDependencies()
}

func main() {
	app := fiber.New(fiber.Config{})

	app.Get("/ping", func(c fiber.Ctx) error {
		return c.SendString("Pong")
	})

	err := app.Listen(":" + cfg.Server.Port)
	if err != nil {
		shutdownApp(err, "Failed to start server")
	}
}

func initDependencies() {
	c := sync.Once{}
	c.Do(func() {
		log.Println("Initializing XPTO")
		//...
	})
}

func shutdownApp(err error, message string) {
	if err != nil {
		fmt.Println(message + " - shutdown with error: " + err.Error())
		os.Exit(1)
	}
}
