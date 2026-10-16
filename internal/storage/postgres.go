package storage

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, err
	}
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnLifetime = time.Hour

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for {
		if err := pool.Ping(pingCtx); err == nil {
			return pool, nil
		}
		if pingCtx.Err() != nil {
			pool.Close()
			return nil, pingCtx.Err()
		}
		time.Sleep(500 * time.Millisecond)
	}
}
