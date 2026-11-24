package httpapi

import (
	"net/http"

	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/user/dto/request"
	"e-bank/backend/internal/user/dto/response"
	"e-bank/backend/internal/user/repositories"
	"e-bank/backend/internal/user/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgxpool"
)

func registerReportingRoutes(router chi.Router, pool *pgxpool.Pool, users services.UserCounter) {
	service := services.NewReportingService(repositories.NewReportingRepository(pool), users)
	admin := router.With(auth.RequireRole("ADMIN"))
	admin.Get("/admin/analytics", analyticsHandler(service))
	admin.Get("/admin/audit", auditHandler(service))
}

func analyticsHandler(service *services.ReportingService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		from, to, err := request.ParseAnalytics(r.URL.RawQuery)
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
		filter, err := request.ParseAudit(r.URL.RawQuery)
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
