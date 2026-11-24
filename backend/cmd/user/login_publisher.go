package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"e-bank/backend/internal/infrastructure/config"
	"e-bank/backend/internal/user/services"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/twmb/franz-go/pkg/kgo"
)

func startLoginPublisher(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool) (func(), error) {
	producer, err := kgo.NewClient(
		kgo.SeedBrokers(config.GetKafkaBrokers()...),
		kgo.AllowAutoTopicCreation(),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.RecordDeliveryTimeout(5*time.Second),
	)
	if err != nil {
		return nil, fmt.Errorf("configure User Kafka producer: %w", err)
	}

	publisherCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		services.RunLoginEventPublisher(publisherCtx, logger, pool, producer)
	}()
	return func() {
		cancel()
		producer.Close()
		<-done
	}, nil
}
