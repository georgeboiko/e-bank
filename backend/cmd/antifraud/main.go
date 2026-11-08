package main

import (
	"context"
	"e-bank/backend/internal/antifraud/httpapi"
	"e-bank/backend/internal/infrastructure/httpserver"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	addr := os.Getenv("ANTIFRAUD_HTTP_ADDR")
	if addr == "" {
		addr = "localhost:8082"
	}

	return httpserver.Run(ctx, logger, addr, httpapi.NewRouter())
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "antifraud")

	if err := run(logger); err != nil {
		logger.Error("Service failed", "error", err)
		os.Exit(1)
	}
}
