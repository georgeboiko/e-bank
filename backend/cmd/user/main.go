package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func liveHandler(w http.ResponseWriter, r *http.Request) {
	render.JSON(w, r, map[string]string{
		"status": "UP",
	})
}

func run(logger *slog.Logger) error {
	addr := os.Getenv("USER_HTTP_ADDR")
	if addr == "" {
		addr = "localhost:8080"
	}

	router := chi.NewRouter()
	router.Get("/live", liveHandler)

	server := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	ctx, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	serverErrors := make(chan error, 1)

	go func() {
		logger.Info("Starting HTTP server", "addr", addr)
		serverErrors <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErrors:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("HTTP server: %w", err)
	case <-ctx.Done():
		logger.Info("Stopping HTTP server")
	}

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		_ = server.Close()
		return fmt.Errorf("HTTP server shutdown: %w", err)
	}

	err := <-serverErrors
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("HTTP server stopped: %w", err)
	}

	logger.Info("HTTP server stopped")
	return nil

}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).With("service", "user")

	if err := run(logger); err != nil {
		logger.Error("Service failed", "error", err)
		os.Exit(1)
	}
}
