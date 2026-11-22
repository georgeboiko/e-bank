package httpapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/money/dto/request"
	"e-bank/backend/internal/money/dto/response"
	"e-bank/backend/internal/money/models"
	"e-bank/backend/internal/money/services"

	"github.com/go-chi/chi/v5"
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
			render.JSON(w, r, toAccountResponse(account))
		}
	}
}

func listAccountsHandler(accountService *services.AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"Некорректный запрос", "Проверьте параметры запроса.")
			return
		}

		principal, _ := auth.PrincipalFromContext(r.Context())
		admin := principal.HasRole("ADMIN")
		if query.Has("ownerId") && !admin {
			httpserver.WriteProblem(w, http.StatusForbidden, "FORBIDDEN",
				"Запрещённый фильтр", "Фильтр ownerId доступен только администратору.")
			return
		}
		params, err := request.ParseAccountList(query)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", err.Error())
			return
		}

		accounts, hasMore, err := accountService.ListAccounts(r.Context(), principal.Subject, admin, models.AccountListFilter(params))
		if err != nil {
			httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка", "Не удалось получить список счетов.")
			return
		}

		items := make([]response.Account, 0, len(accounts))
		for _, account := range accounts {
			items = append(items, toAccountResponse(account))
		}
		render.JSON(w, r, response.AccountPage{
			Items:   items,
			Limit:   params.Limit,
			Offset:  params.Offset,
			HasMore: hasMore,
		})
	}
}

func getAccountHandler(accountService *services.AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "accountId"), 10, 32)
		if err != nil || id < 1 {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"Некорректный запрос", "accountId должен быть положительным целым числом типа int32.")
			return
		}

		principal, _ := auth.PrincipalFromContext(r.Context())
		account, err := accountService.GetAccount(r.Context(), principal.Subject, principal.HasRole("ADMIN"), int32(id))
		switch {
		case errors.Is(err, models.ErrAccountNotFound):
			httpserver.WriteProblem(w, http.StatusNotFound, "NOT_FOUND", "Счёт не найден", "Запрошенный счёт недоступен.")
		case err != nil:
			httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка", "Не удалось получить счёт.")
		default:
			render.JSON(w, r, toAccountResponse(account))
		}
	}
}

func toAccountResponse(account models.Account) response.Account {
	return response.Account{
		ID:               account.ID,
		OwnerID:          account.OwnerID,
		Type:             account.Type,
		State:            account.State,
		Balance:          account.Balance,
		ReservedSum:      account.ReservedSum,
		AvailableBalance: account.AvailableBalance,
		OpenedAt:         account.OpenedAt.UTC(),
	}
}
