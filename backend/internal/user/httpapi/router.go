package httpapi

import (
	"net/http"

	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/user/keycloak"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(pool *pgxpool.Pool, authenticate func(http.Handler) http.Handler, directory *keycloak.Client) http.Handler {
	router := chi.NewRouter()
	registerAdminReadRoutes(router, directory)

	return httpserver.WithHealth(pool, authenticate(router))
}
