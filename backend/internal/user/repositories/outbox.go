package repositories

import (
	"context"

	"e-bank/backend/internal/contracts/kafka"
	"e-bank/backend/internal/infrastructure/db/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ReadOutboxEvent(ctx context.Context, pool *pgxpool.Pool, eventID string) (postgres.OutboxEvent[kafka.LoginSucceededData], error) {
	return postgres.ReadOutboxEvent[kafka.LoginSucceededData](ctx, pool, eventID, kafka.EventLoginSucceeded)
}
