package repositories

import (
	"context"

	"e-bank/backend/internal/contracts/kafka"
	"e-bank/backend/internal/infrastructure/db/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ReadOutboxEvent(ctx context.Context, pool *pgxpool.Pool, eventID string) (postgres.OutboxEvent[kafka.FraudCheckRequestedData], error) {
	return postgres.ReadOutboxEvent[kafka.FraudCheckRequestedData](ctx, pool, eventID, kafka.EventFraudCheckRequested)
}
