package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"e-bank/backend/internal/antifraud/dto/request"
	"e-bank/backend/internal/antifraud/dto/response"
	"e-bank/backend/internal/antifraud/models"
	"e-bank/backend/internal/antifraud/services"
	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/infrastructure/httpserver"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgxpool"
)

func registerReportingRoutes(router chi.Router, pool *pgxpool.Pool) {
	service := services.NewReportingService(pool)
	router.With(auth.RequireRole("ANALYST", "ADMIN")).Get("/analytics", analyticsHandler(service))
	router.With(auth.RequireRole("ADMIN")).Get("/audit", auditHandler(service, false))
	router.With(auth.RequireRole("ANALYST", "ADMIN")).Get("/checks/{checkId}/audit", auditHandler(service, true))
}

func analyticsHandler(service *services.ReportingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, to, topLimit, err := request.ParseAnalytics(r.URL.RawQuery)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", err.Error())
			return
		}
		data, err := service.Analytics(r.Context(), from, to, topLimit)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка", "Не удалось получить аналитику.")
			return
		}
		render.JSON(w, r, response.FromAnalytics(data))
	}
}

func auditHandler(service *services.ReportingService, forCheck bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var checkID *int32
		if forCheck {
			id, err := strconv.ParseInt(chi.URLParam(r, "checkId"), 10, 32)
			if err != nil || id < 1 {
				httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос",
					"checkId должен быть положительным целым числом.")
				return
			}
			value := int32(id)
			checkID = &value
		}
		filter, err := request.ParseAudit(r.URL.RawQuery, checkID)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", err.Error())
			return
		}
		events, err := service.Audit(r.Context(), filter, forCheck)
		switch {
		case errors.Is(err, models.ErrAuditCheckNotFound):
			httpserver.WriteProblem(w, http.StatusNotFound, "NOT_FOUND", "Проверка не найдена", "Проверьте checkId.")
		case err != nil:
			httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка", "Не удалось прочитать аудит.")
		default:
			render.JSON(w, r, response.ToAuditPage(events, filter.Limit, filter.Offset))
		}
	}
}
