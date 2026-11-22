package services

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"e-bank/backend/internal/contracts/kafka"
	"e-bank/backend/internal/money/repositories"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
)

func RunFraudRequestPublisher(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool, client *kgo.Client) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	publish := func(transferID int32, payload []byte) error {
		return client.ProduceSync(ctx, &kgo.Record{
			Topic: kafka.TopicFraudCheckRequested,
			Key:   []byte(strconv.FormatInt(int64(transferID), 10)),
			Value: payload,
		}).FirstErr()
	}

	for ctx.Err() == nil {
		attempted, err := repositories.PublishNextOutboxEvent(ctx, pool, publish)
		if err != nil && ctx.Err() == nil {
			logger.Warn("Money outbox publish attempt failed", "error", err)
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
