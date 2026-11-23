package httpapi

import (
	"errors"
	"net/http"
	"net/url"

	"e-bank/backend/internal/antifraud/dto/request"
	"e-bank/backend/internal/antifraud/dto/response"
	"e-bank/backend/internal/antifraud/models"
	"e-bank/backend/internal/antifraud/repositories"
	"e-bank/backend/internal/antifraud/services"
	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/infrastructure/httpserver"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgxpool"
)

func registerReviewQueueRoutes(router chi.Router, pool *pgxpool.Pool) {
	service := services.NewReviewQueueService(repositories.NewReviewQueueRepository(pool))
	read := router.With(auth.RequireRole("ANALYST", "ADMIN"))
	read.Get("/checks", listChecksHandler(service))
	read.Get("/checks/{checkId}", getCheckHandler(service))
	read.Get("/review-cases", listReviewCasesHandler(service))
	read.Get("/review-cases/{caseId}", getReviewCaseHandler(service))
}

func listChecksHandler(service *services.ReviewQueueService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			writeReviewQueueValidationError(w, "Проверьте параметры запроса.")
			return
		}
		filter, err := request.ParseCheckList(query)
		if err != nil {
			writeReviewQueueValidationError(w, err.Error())
			return
		}
		checks, hasMore, err := service.ListChecks(r.Context(), filter)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка", "Не удалось получить список проверок.")
			return
		}
		items := make([]response.CheckSummary, 0, len(checks))
		for _, check := range checks {
			items = append(items, response.FromCheckSummary(check))
		}
		render.JSON(w, r, response.CheckPage{Items: items, Limit: filter.Limit, Offset: filter.Offset, HasMore: hasMore})
	}
}

func getCheckHandler(service *services.ReviewQueueService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := request.ParseEntityID("checkId", chi.URLParam(r, "checkId"))
		if err != nil {
			writeReviewQueueValidationError(w, err.Error())
			return
		}
		check, err := service.GetCheck(r.Context(), id)
		switch {
		case errors.Is(err, models.ErrCheckNotFound):
			httpserver.WriteProblem(w, http.StatusNotFound, "NOT_FOUND",
				"Проверка не найдена", "Запрошенная проверка недоступна.")
		case err != nil:
			httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка", "Не удалось получить проверку.")
		default:
			render.JSON(w, r, response.FromCheck(check))
		}
	}
}

func listReviewCasesHandler(service *services.ReviewQueueService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			writeReviewQueueValidationError(w, "Проверьте параметры запроса.")
			return
		}
		filter, err := request.ParseReviewCaseList(query)
		if err != nil {
			writeReviewQueueValidationError(w, err.Error())
			return
		}
		reviews, hasMore, err := service.ListReviewCases(r.Context(), filter)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка", "Не удалось получить очередь рассмотрений.")
			return
		}
		items := make([]response.ReviewCase, 0, len(reviews))
		for _, review := range reviews {
			items = append(items, response.FromReviewCase(review))
		}
		render.JSON(w, r, response.ReviewPage{Items: items, Limit: filter.Limit, Offset: filter.Offset, HasMore: hasMore})
	}
}

func getReviewCaseHandler(service *services.ReviewQueueService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := request.ParseEntityID("caseId", chi.URLParam(r, "caseId"))
		if err != nil {
			writeReviewQueueValidationError(w, err.Error())
			return
		}
		review, err := service.GetReviewCase(r.Context(), id)
		switch {
		case errors.Is(err, models.ErrReviewCaseNotFound):
			httpserver.WriteProblem(w, http.StatusNotFound, "NOT_FOUND",
				"Карточка не найдена", "Запрошенная карточка ручного рассмотрения недоступна.")
		case err != nil:
			httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка", "Не удалось получить карточку ручного рассмотрения.")
		default:
			render.JSON(w, r, response.FromReviewCase(review))
		}
	}
}

func writeReviewQueueValidationError(w http.ResponseWriter, detail string) {
	httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", detail)
}
