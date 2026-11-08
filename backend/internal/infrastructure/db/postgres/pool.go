package postgres

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(
	ctx context.Context,
	datasource string,
) (*pgxpool.Pool, error) {
	if datasource == "" {
		return nil, errors.New("datasource is empty")
	}

	pool, err := pgxpool.New(ctx, datasource)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(
		ctx, 5*time.Second,
	)
	defer cancel()

	err = pool.Ping(pingCtx)
	if err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping PostgreSQL pool: %w", err)
	}

	return pool, nil
}
