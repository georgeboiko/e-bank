package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/money/models"
	"e-bank/backend/internal/money/services"

	"github.com/go-chi/render"
)

func openAccountHandler(accountService *services.AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Body != nil && r.Body != http.NoBody {
			body, err := io.ReadAll(io.LimitReader(r.Body, 1))
			if err != nil || len(body) != 0 {
				writeProblem(w, http.StatusBadRequest, "VALIDATION_ERROR",
					"Некорректный запрос", "Для открытия счёта требуется пустое тело запроса.")
				return
			}
		}

		principal, _ := auth.PrincipalFromContext(r.Context())
		account, err := accountService.OpenAccount(r.Context(), principal.Subject)
		switch {
		case errors.Is(err, models.ErrAccountLimitReached):
			writeProblem(w, http.StatusConflict, "ACCOUNT_LIMIT_REACHED",
				"Достигнут лимит счетов", "Нельзя открыть ещё один счёт.")
		case err != nil:
			writeProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка", "Не удалось открыть счёт.")
		default:
			account.OpenedAt = account.OpenedAt.UTC()
			w.Header().Set("Location", fmt.Sprintf("/api/money/v1/accounts/%d", account.ID))
			render.Status(r, http.StatusCreated)
			render.JSON(w, r, account)
		}
	}
}

type problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Code   string `json:"code"`
	Detail string `json:"detail"`
}

func writeProblem(w http.ResponseWriter, status int, code, title, detail string) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(problem{
		Type:   "urn:ebank:problem:" + strings.ToLower(strings.ReplaceAll(code, "_", "-")),
		Title:  title,
		Status: status,
		Code:   code,
		Detail: detail,
	})
}
