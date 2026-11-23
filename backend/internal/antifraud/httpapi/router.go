package httpapi

import (
	"net/http"

	"e-bank/backend/internal/infrastructure/httpserver"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(pool *pgxpool.Pool, authenticate func(http.Handler) http.Handler) http.Handler {
	router := chi.NewRouter()
	registerReviewQueueRoutes(router, pool)
	registerReviewGraphRoutes(router, pool)
	registerReviewDecisionRoutes(router, pool)

	return httpserver.WithHealth(pool, authenticate(router))
}
