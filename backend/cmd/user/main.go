package main

import (
	"context"
	"e-bank/backend/internal/infrastructure/httpserver"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-chi/chi/v5"
)

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	addr := os.Getenv("USER_HTTP_ADDR")
	if addr == "" {
		addr = "localhost:8080"
	}

	router := chi.NewRouter()
	router.Get("/live", httpserver.LiveHandler)

	return httpserver.Run(ctx, logger, addr, router)
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "user")

	if err := run(logger); err != nil {
		logger.Error("Service failed", "error", err)
		os.Exit(1)
	}
}
