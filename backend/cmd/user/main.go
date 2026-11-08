package main

import (
	"e-bank/backend/internal/infrastructure/config"
	"e-bank/backend/internal/infrastructure/db/postgres"
	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/infrastructure/util"
	"e-bank/backend/internal/user/httpapi"
	"log/slog"
	"os"
)

const (
	Service = "USER"
	Port    = 8080
)

func run(logger *slog.Logger) error {
	ctx, stop := util.NotifyShutdown()
	defer stop()

	addr := config.GetAddr(Service, Port)
	datasource, err := config.GetDatasource(Service)
	if err != nil {
		return err
	}

	pool, err := postgres.NewPool(ctx, datasource)
	if err != nil {
		return err
	}
	defer pool.Close()

	return httpserver.Run(ctx, logger, addr, httpapi.NewRouter())
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "user")

	if err := run(logger); err != nil {
		logger.Error("Service failed", "error", err)
		os.Exit(1)
	}
}
