package httpapi

import (
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

func registerRuleSettingsRoutes(router chi.Router, pool *pgxpool.Pool) {
	service := services.NewRuleSettingsService(pool)
	router.With(auth.RequireRole("ANALYST", "ADMIN")).Get("/rules", listRulesHandler(service))
	router.With(auth.RequireRole("ADMIN")).Put("/rules/{ruleId}", updateRuleHandler(service))
}

func listRulesHandler(service *services.RuleSettingsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		configs, err := service.List(r.Context())
		if err != nil {
			httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка", "Не удалось получить настройки антифрода.")
			return
		}
		result := make([]response.RuleConfig, 0, len(configs))
		for _, config := range configs {
			result = append(result, response.ToRuleConfig(config))
		}
		render.JSON(w, r, result)
	}
}

func updateRuleHandler(service *services.RuleSettingsService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "ruleId"), 10, 32)
		if err != nil || id < 1 || id > 8 {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"Некорректный запрос", "ruleId должен быть от 1 до 8.")
			return
		}
		rule := models.DefaultRuleConfigs()[id-1]
		settings, err := request.ParseRuleSettings(http.MaxBytesReader(w, r.Body, 64<<10), rule)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", err.Error())
			return
		}
		principal, _ := auth.PrincipalFromContext(r.Context())
		config, err := service.Update(r.Context(), principal.Subject, rule, settings)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка", "Не удалось сохранить настройку правила.")
			return
		}
		render.JSON(w, r, response.ToRuleConfig(config))
	}
}
