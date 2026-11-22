package repositories

import (
	"context"

	"e-bank/backend/internal/contracts/kafka"
	"e-bank/backend/internal/infrastructure/db/postgres"

	"github.com/jackc/pgx/v5/pgxpool"
)

func ReadOutboxEvent(ctx context.Context, pool *pgxpool.Pool, eventID string) (postgres.OutboxEvent[kafka.FraudDecisionData], error) {
	return postgres.ReadOutboxEvent[kafka.FraudDecisionData](ctx, pool, eventID, kafka.EventFraudDecision)
}
