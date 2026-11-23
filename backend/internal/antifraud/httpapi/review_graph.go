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

func registerReviewGraphRoutes(router chi.Router, pool *pgxpool.Pool) {
	service := services.NewReviewGraphService(repositories.NewReviewGraphRepository(pool))
	router.With(auth.RequireRole("ANALYST", "ADMIN")).Get("/graph", getReviewGraphHandler(service))
}

func getReviewGraphHandler(service *services.ReviewGraphService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		values, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"Некорректный запрос", "Параметры графа заданы неверно.")
			return
		}
		query, err := request.ParseGraphQuery(values)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR",
				"Некорректный запрос", err.Error())
			return
		}

		graph, err := service.Get(r.Context(), query)
		switch {
		case errors.Is(err, models.ErrGraphRootNotFound):
			httpserver.WriteProblem(w, http.StatusNotFound, "NOT_FOUND",
				"Корень графа не найден", "Запрошенная проверка или объект отсутствует.")
		case err != nil:
			httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка", "Не удалось получить граф связей.")
		default:
			render.JSON(w, r, response.NewGraph(graph))
		}
	}
}
