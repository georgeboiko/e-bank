package httpapi

import (
	"net/http"

	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/money/services"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(pool *pgxpool.Pool, authenticate func(http.Handler) http.Handler, accountService *services.AccountService) http.Handler {
	router := chi.NewRouter()
	router.With(auth.RequireRole("CLIENT")).Post("/accounts", openAccountHandler(accountService))

	return httpserver.WithHealth(pool, authenticate(router))
}
