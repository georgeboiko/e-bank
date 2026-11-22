package services

import (
	"context"
	"log/slog"
	"time"

	"e-bank/backend/internal/contracts/kafka"
	"e-bank/backend/internal/user/repositories"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
)

func RunLoginEventPublisher(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool, client *kgo.Client) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	publish := func(userID string, payload []byte) error {
		return client.ProduceSync(ctx, &kgo.Record{
			Topic: kafka.TopicLoginSucceeded,
			Key:   []byte(userID),
			Value: payload,
		}).FirstErr()
	}

	for ctx.Err() == nil {
		attempted, err := repositories.PublishNextOutboxEvent(ctx, pool, publish)
		if err != nil && ctx.Err() == nil {
			logger.Warn("User outbox publish attempt failed", "error", err)
		}
		if attempted && err == nil {
			continue
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}
