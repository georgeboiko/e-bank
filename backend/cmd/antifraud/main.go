package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"e-bank/backend/internal/antifraud/httpapi"
	"e-bank/backend/internal/antifraud/services"
	"e-bank/backend/internal/contracts/kafka"
	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/infrastructure/config"
	"e-bank/backend/internal/infrastructure/db/postgres"
	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/infrastructure/util"

	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	Service = "ANTIFRAUD"
	Port    = 8082
)

func run(logger *slog.Logger) error {
	ctx, stop := util.NotifyShutdown()
	defer stop()

	addr := config.GetAddr(Service, Port)
	authenticate, err := auth.NewJWTMiddleware(os.Getenv("KEYCLOAK_ISSUER"), os.Getenv("KEYCLOAK_AUDIENCE"))
	if err != nil {
		return err
	}

	datasource, err := config.GetDatasource(Service)
	if err != nil {
		return err
	}

	pool, err := postgres.NewPool(ctx, datasource)
	if err != nil {
		return err
	}
	defer pool.Close()

	consumer, err := kgo.NewClient(
		kgo.SeedBrokers(config.GetKafkaBrokers()...),
		kgo.ConsumeTopics(kafka.TopicFraudCheckRequested),
		kgo.ConsumerGroup("ebank-antifraud-fraud-requests"),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(),
		kgo.BlockRebalanceOnPoll(),
	)
	if err != nil {
		return fmt.Errorf("configure Antifraud Kafka consumer: %w", err)
	}
	defer consumer.CloseAllowingRebalance()

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- services.RunFraudRequestConsumer(runCtx, logger, pool, consumer)
		cancel()
	}()

	serverErr := httpserver.Run(runCtx, logger, addr, httpapi.NewRouter(pool, authenticate))
	cancel()
	return errors.Join(serverErr, <-consumerDone)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "antifraud")

	if err := run(logger); err != nil {
		logger.Error("Service failed", "error", err)
		os.Exit(1)
	}
}
