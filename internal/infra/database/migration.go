package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

func RunMigrations(db *pgxpool.Pool) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS supplies (
			id         UUID    PRIMARY KEY DEFAULT gen_random_uuid(),
			service_id UUID    NOT NULL,
			item_id    UUID    NOT NULL,
			quantity   INTEGER NOT NULL
		)`,
	}

	for i, migration := range migrations {
		if _, err := db.Exec(context.Background(), migration); err != nil {
			return fmt.Errorf("migration %d failed: %w", i+1, err)
		}
	}

	return nil
}
