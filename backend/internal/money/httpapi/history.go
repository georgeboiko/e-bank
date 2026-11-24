package httpapi

import (
	"errors"
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

func listTransfersHandler(service *services.TransferService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"Некорректный запрос", "Проверьте параметры запроса.")
			return
		}

		principal, _ := auth.PrincipalFromContext(r.Context())
		admin := principal.HasRole("ADMIN")
		if query.Has("userId") && !admin {
			httpserver.WriteProblem(w, http.StatusForbidden, "FORBIDDEN",
				"Запрещённый фильтр", "Фильтр userId доступен только администратору.")
			return
		}
		filter, err := request.ParseTransferList(query)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", err.Error())
			return
		}

		transfers, hasMore, err := service.ListTransfers(r.Context(), principal.Subject, admin, filter)
		if err != nil {
			writeHistoryError(w, err, "Не удалось получить историю переводов.")
			return
		}
		items := make([]response.Transfer, 0, len(transfers))
		for _, transfer := range transfers {
			items = append(items, toTransferResponse(transfer))
		}
		render.JSON(w, r, response.TransferPage{
			Items: items, Limit: filter.Limit, Offset: filter.Offset, HasMore: hasMore,
		})
	}
}

func listLedgerHandler(service *services.AccountService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accountID, err := strconv.ParseInt(chi.URLParam(r, "accountId"), 10, 32)
		if err != nil || accountID < 1 {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"Некорректный запрос", "accountId должен быть положительным целым числом типа int32.")
			return
		}
		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"Некорректный запрос", "Проверьте параметры запроса.")
			return
		}
		filter, err := request.ParseHistory(query)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", err.Error())
			return
		}

		principal, _ := auth.PrincipalFromContext(r.Context())
		entries, hasMore, err := service.ListLedger(r.Context(), principal.Subject, principal.HasRole("ADMIN"), int32(accountID), filter)
		if err != nil {
			writeHistoryError(w, err, "Не удалось получить проводки счёта.")
			return
		}
		items := make([]response.LedgerEntry, 0, len(entries))
		for _, entry := range entries {
			items = append(items, response.LedgerEntry{
				ID:         entry.ID,
				TransferID: entry.TransferID,
				AccountID:  entry.AccountID,
				Amount:     entry.Amount.StringFixed(2),
				CreatedAt:  entry.CreatedAt.UTC(),
			})
		}
		render.JSON(w, r, response.LedgerPage{
			Items: items, Limit: filter.Limit, Offset: filter.Offset, HasMore: hasMore,
		})
	}
}

func writeHistoryError(w http.ResponseWriter, err error, detail string) {
	if errors.Is(err, models.ErrAccountNotFound) {
		httpserver.WriteProblem(w, http.StatusNotFound, "NOT_FOUND",
			"Счёт не найден", "Запрошенный счёт недоступен.")
		return
	}
	httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка", detail)
}
