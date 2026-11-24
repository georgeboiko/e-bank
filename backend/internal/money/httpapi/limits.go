package httpapi

import (
	"errors"
	"net/http"

	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/money/dto/request"
	"e-bank/backend/internal/money/dto/response"
	"e-bank/backend/internal/money/models"
	"e-bank/backend/internal/money/repositories"
	"e-bank/backend/internal/money/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgxpool"
)

func registerTransferLimitRoutes(router chi.Router, pool *pgxpool.Pool) {
	service := services.NewTransferLimitService(repositories.NewTransferLimitRepository(pool))
	router.With(auth.RequireRole("CLIENT", "ADMIN")).Get("/limits", getTransferLimitsHandler(service))
	router.With(auth.RequireRole("CLIENT")).Get("/limits/usage", getTransferLimitUsageHandler(service))
	router.With(auth.RequireRole("ADMIN")).Put("/limits", updateTransferLimitsHandler(service))
}

func getTransferLimitsHandler(service *services.TransferLimitService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limits, err := service.Get(r.Context())
		if err != nil {
			writeTransferLimitsError(w, err, "Не удалось получить лимиты переводов.")
			return
		}
		render.JSON(w, r, response.FromTransferLimits(limits))
	}
}

func getTransferLimitUsageHandler(service *services.TransferLimitService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		principal, _ := auth.PrincipalFromContext(r.Context())
		usage, err := service.GetUsage(r.Context(), principal.Subject)
		if err != nil {
			writeTransferLimitsError(w, err, "Не удалось получить расход лимитов переводов.")
			return
		}
		render.JSON(w, r, response.FromTransferLimitUsage(usage))
	}
}

func updateTransferLimitsHandler(service *services.TransferLimitService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		limits, err := request.ParseTransferLimits(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", err.Error())
			return
		}

		principal, _ := auth.PrincipalFromContext(r.Context())
		if err := service.Update(r.Context(), principal.Subject, limits); err != nil {
			writeTransferLimitsError(w, err, "Не удалось сохранить лимиты переводов.")
			return
		}
		render.JSON(w, r, response.FromTransferLimits(limits))
	}
}

func writeTransferLimitsError(w http.ResponseWriter, err error, detail string) {
	if errors.Is(err, models.ErrLimitsNotConfigured) {
		httpserver.WriteProblem(w, http.StatusInternalServerError, "LIMITS_NOT_CONFIGURED",
			"Лимиты переводов не настроены", "Настройка общих лимитов переводов отсутствует.")
		return
	}
	httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка", detail)
}
