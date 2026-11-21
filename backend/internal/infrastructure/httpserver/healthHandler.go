package httpserver

import (
	"context"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	pool *pgxpool.Pool
}

func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{pool: pool}
}

func WithHealth(pool *pgxpool.Pool, handler http.Handler) http.Handler {
	router := chi.NewRouter()
	router.Get("/health", NewHealthHandler(pool).ServeHTTP)
	router.Mount("/", handler)
	return router
}

func (h *HealthHandler) ServeHTTP(
	w http.ResponseWriter,
	r *http.Request,
) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.pool.Ping(ctx); err != nil {
		render.Status(r, http.StatusServiceUnavailable)
		render.JSON(w, r, map[string]string{
			"status": "DOWN",
		})
		return
	}

	render.JSON(w, r, map[string]string{
		"status": "UP",
	})
}
