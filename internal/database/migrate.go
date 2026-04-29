package database

import (
	"log"

	dbmigrations "github.com/ESSantana/15soat-tech-challenge-step-1/database"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	_ "github.com/lib/pq"
	"github.com/pressly/goose/v3"
)

func RunMigrations(pool *pgxpool.Pool) {
	goose.SetBaseFS(dbmigrations.Migrations)

	if err := goose.SetDialect("postgres"); err != nil {
		log.Fatalf("Failed to set goose dialect: %v", err)
	}

	db := stdlib.OpenDBFromPool(pool)
	defer db.Close()

	if err := goose.Up(db, "migrations"); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Migrations applied successfully")
}
