package httpapi

import (
	"e-bank/backend/internal/infrastructure/httpserver"
	"net/http"

	"github.com/go-chi/chi/v5"
)

func NewRouter() http.Handler {
	router := chi.NewRouter()
	router.Get("/live", httpserver.LiveHandler)

	return router
}
