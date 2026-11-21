package httpapi

import (
	"net/http"

	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/money/services"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(pool *pgxpool.Pool, accountService *services.AccountService) http.Handler {
	router := chi.NewRouter()
	router.Get("/health", httpserver.NewHealthHandler(pool).ServeHTTP)
	router.With(auth.RequireRole("CLIENT")).Post("/accounts", openAccountHandler(accountService))

	return router
}
