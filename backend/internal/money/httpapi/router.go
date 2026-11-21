package httpapi

import (
	"e-bank/backend/internal/infrastructure/httpserver"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(pool *pgxpool.Pool) http.Handler {
	router := chi.NewRouter()
	router.Get("/health", httpserver.NewHealthHandler(pool).ServeHTTP)

	return router
}
