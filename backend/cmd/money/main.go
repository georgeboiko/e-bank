package main

import (
	"log/slog"
	"os"

	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/infrastructure/config"
	"e-bank/backend/internal/infrastructure/db/postgres"
	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/infrastructure/util"
	"e-bank/backend/internal/money/httpapi"
	"e-bank/backend/internal/money/repositories"
	"e-bank/backend/internal/money/services"
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

	accountRepository := repositories.NewAccountRepository(pool)
	accountService := services.NewAccountService(accountRepository, config.MoneyMaxAccountsPerClient)
	transferRepository := repositories.NewTransferRepository(pool)
	transferService := services.NewTransferService(transferRepository)
	return httpserver.Run(ctx, logger, addr, httpapi.NewRouter(pool, authenticate, accountService, transferService))
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "money")

	if err := run(logger); err != nil {
		logger.Error("Service failed", "error", err)
		os.Exit(1)
	}
}
