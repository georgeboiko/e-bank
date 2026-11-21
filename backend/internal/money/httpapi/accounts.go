package httpapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"

	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/money/dto/response"
	"e-bank/backend/internal/money/models"
	"e-bank/backend/internal/money/services"

	"github.com/go-chi/render"
)

func openAccountHandler(accountService *services.AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.Body != http.NoBody {
			body, err := io.ReadAll(io.LimitReader(r.Body, 1))
			if err != nil || len(body) != 0 {
				httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR",
					"Некорректный запрос", "Для открытия счёта требуется пустое тело запроса.")
				return
			}
		}

		principal, _ := auth.PrincipalFromContext(r.Context())
		account, err := accountService.OpenAccount(r.Context(), principal.Subject)
		switch {
		case errors.Is(err, models.ErrAccountLimitReached):
			httpserver.WriteProblem(w, http.StatusConflict, "ACCOUNT_LIMIT_REACHED",
				"Достигнут лимит счетов", "Нельзя открыть ещё один счёт.")
		case err != nil:
			httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка", "Не удалось открыть счёт.")
		default:
			w.Header().Set("Location", fmt.Sprintf("/api/money/v1/accounts/%d", account.ID))
			render.Status(r, http.StatusCreated)
			render.JSON(w, r, response.Account{
				ID:               account.ID,
				OwnerID:          account.OwnerID,
				Type:             account.Type,
				State:            account.State,
				Balance:          account.Balance,
				ReservedSum:      account.ReservedSum,
				AvailableBalance: account.AvailableBalance,
				OpenedAt:         account.OpenedAt.UTC(),
			})
		}
	}
}
