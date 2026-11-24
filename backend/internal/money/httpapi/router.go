package httpapi

import (
	"net/http"

	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/money/services"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewRouter(
	pool *pgxpool.Pool,
	authenticate func(http.Handler) http.Handler,
	accountService *services.AccountService,
	transferService *services.TransferService,
	topupService *services.TopupService,
) http.Handler {
	router := chi.NewRouter()
	registerTransferLimitRoutes(router, pool)
	client := router.With(auth.RequireRole("CLIENT"))
	client.Post("/accounts", openAccountHandler(accountService))
	client.Post("/transfers", createTransferHandler(transferService))
	client.Post("/topups", createTopupHandler(topupService))
	client.Post("/topups/{topupId}/refresh", refreshTopupHandler(topupService))
	read := router.With(auth.RequireRole("CLIENT", "ADMIN"))
	read.Get("/accounts", listAccountsHandler(accountService))
	read.Get("/accounts/{accountId}", getAccountHandler(accountService))
	read.Get("/accounts/{accountId}/ledger", listLedgerHandler(accountService))
	read.Get("/transfers", listTransfersHandler(transferService))
	read.Get("/transfers/{transferId}", getTransferHandler(transferService))
	read.Get("/topups/{topupId}", getTopupHandler(topupService))

	return httpserver.WithHealth(pool, authenticate(router))
}
