package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"

	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/user/services"

	"github.com/jackc/pgx/v5/pgxpool"
)

func runWithLoginIngestion(ctx context.Context, logger *slog.Logger, pool *pgxpool.Pool, addr string, handler http.Handler) error {
	clientID := os.Getenv("KEYCLOAK_EVENTS_CLIENT_ID")
	secret := os.Getenv("KEYCLOAK_EVENTS_CLIENT_SECRET")
	if clientID == "" && secret == "" {
		return httpserver.Run(ctx, logger, addr, handler)
	}
	loginClient := os.Getenv("KEYCLOAK_LOGIN_CLIENT_ID")
	if clientID == "" || secret == "" || loginClient == "" {
		return errors.New("Keycloak login ingestion requires KEYCLOAK_EVENTS_CLIENT_ID, KEYCLOAK_EVENTS_CLIENT_SECRET and KEYCLOAK_LOGIN_CLIENT_ID")
	}

	source, err := services.NewKeycloakLoginSource(
		os.Getenv("KEYCLOAK_ISSUER"), os.Getenv("KEYCLOAK_EVENTS_REALM_URL"), clientID, secret, loginClient,
	)
	if err != nil {
		return err
	}

	loginCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		source.Run(loginCtx, logger, pool)
	}()
	defer func() {
		cancel()
		<-done
	}()

	return httpserver.Run(ctx, logger, addr, handler)
}
