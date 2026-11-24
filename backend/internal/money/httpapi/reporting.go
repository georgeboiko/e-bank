package httpapi

import (
	"net/http"
	"net/url"

	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/money/dto/request"
	"e-bank/backend/internal/money/dto/response"
	"e-bank/backend/internal/money/repositories"
	"e-bank/backend/internal/money/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgxpool"
)

func registerReportingRoutes(router chi.Router, pool *pgxpool.Pool) {
	repository := repositories.NewReportingRepository(pool)
	service := services.NewReportingService(repository)
	admin := router.With(auth.RequireRole("ADMIN"))
	admin.Get("/analytics", analyticsHandler(service))
	admin.Get("/audit", auditHandler(service))
}

func analyticsHandler(service *services.ReportingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"Некорректный запрос", "Проверьте параметры запроса.")
			return
		}
		from, to, err := request.ParseAnalytics(query)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", err.Error())
			return
		}
		data, err := service.Analytics(r.Context(), from, to)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка", "Не удалось получить аналитику.")
			return
		}
		render.JSON(w, r, response.FromAnalytics(data))
	}
}

func auditHandler(service *services.ReportingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"Некорректный запрос", "Проверьте параметры запроса.")
			return
		}
		filter, err := request.ParseAudit(query)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", err.Error())
			return
		}
		events, hasMore, err := service.Audit(r.Context(), filter)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка", "Не удалось получить аудит.")
			return
		}
		render.JSON(w, r, response.ToAuditPage(events, filter.Limit, filter.Offset, hasMore))
	}
}
