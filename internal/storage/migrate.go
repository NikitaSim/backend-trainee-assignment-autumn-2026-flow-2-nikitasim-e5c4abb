package storage

import (
	"context"
	"embed"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/001_init.up.sql
var migrations embed.FS

func Migrate(ctx context.Context, pool *pgxpool.Pool) error {
	const createTable = `
        CREATE TABLE IF NOT EXISTS schema_migrations (
            version BIGINT PRIMARY KEY,
            applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
        )`
	if _, err := pool.Exec(ctx, createTable); err != nil {
		return err
	}

	var applied bool
	err := pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version=1)`).Scan(&applied)
	if err != nil {
		return err
	}
	if applied {
		return nil
	}

	migration, err := migrations.ReadFile("migrations/001_init.up.sql")
	if err != nil {
		return err
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, string(migration)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO schema_migrations(version) VALUES(1)`); err != nil {
		return err
	}

	return tx.Commit(ctx)
}
