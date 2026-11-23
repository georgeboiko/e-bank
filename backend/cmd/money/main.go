package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"e-bank/backend/internal/contracts/kafka"
	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/infrastructure/config"
	"e-bank/backend/internal/infrastructure/db/postgres"
	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/infrastructure/util"
	"e-bank/backend/internal/money/httpapi"
	"e-bank/backend/internal/money/repositories"
	"e-bank/backend/internal/money/services"

	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	Service = "MONEY"
	Port    = 8081
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
		kgo.ConsumeTopics(kafka.TopicFraudDecision),
		kgo.ConsumerGroup("ebank-money-fraud-decisions"),
		kgo.ConsumeResetOffset(kgo.NewOffset().AtStart()),
		kgo.DisableAutoCommit(),
		kgo.BlockRebalanceOnPoll(),
	)
	if err != nil {
		return fmt.Errorf("configure Money Kafka consumer: %w", err)
	}
	defer consumer.CloseAllowingRebalance()

	producer, err := kgo.NewClient(
		kgo.SeedBrokers(config.GetKafkaBrokers()...),
		kgo.AllowAutoTopicCreation(),
		kgo.RequiredAcks(kgo.AllISRAcks()),
		kgo.RecordDeliveryTimeout(5*time.Second),
	)
	if err != nil {
		return fmt.Errorf("configure Money Kafka producer: %w", err)
	}
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	publisherDone := make(chan struct{})
	go func() {
		defer close(publisherDone)
		services.RunFraudRequestPublisher(runCtx, logger, pool, producer)
	}()
	defer func() {
		cancel()
		producer.Close()
		<-publisherDone
	}()

	consumerDone := make(chan error, 1)
	go func() {
		consumerDone <- services.RunFraudDecisionConsumer(runCtx, logger, pool, consumer)
		cancel()
	}()

	finalizerDone := make(chan struct{})
	go func() {
		defer close(finalizerDone)
		services.RunTransferFinalizer(runCtx, logger, pool)
	}()
	defer func() {
		cancel()
		<-finalizerDone
	}()

	accountRepository := repositories.NewAccountRepository(pool)
	accountService := services.NewAccountService(accountRepository, config.MoneyMaxAccountsPerClient)
	transferRepository := repositories.NewTransferRepository(pool)
	transferService := services.NewTransferService(transferRepository)
	serverErr := httpserver.Run(runCtx, logger, addr, httpapi.NewRouter(pool, authenticate, accountService, transferService))
	cancel()
	return errors.Join(serverErr, <-consumerDone)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "money")

	if err := run(logger); err != nil {
		logger.Error("Service failed", "error", err)
		os.Exit(1)
	}
}
