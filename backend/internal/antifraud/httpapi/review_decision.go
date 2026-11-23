package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strconv"

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

func registerReviewDecisionRoutes(router chi.Router, pool *pgxpool.Pool) {
	service := services.NewReviewDecisionService(repositories.NewReviewDecisionRepository(pool))
	analyst := router.With(auth.RequireRole("ANALYST"))
	analyst.Post("/review-cases/{caseId}/claim", claimReviewCaseHandler(service))
	analyst.Post("/review-cases/{caseId}/decision", resolveReviewCaseHandler(service))
}

func claimReviewCaseHandler(service *services.ReviewDecisionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caseID, ok := reviewMutationCaseID(w, r)
		if !ok {
			return
		}
		if r.Body != nil && r.Body != http.NoBody {
			body, err := io.ReadAll(io.LimitReader(r.Body, 1))
			if err != nil || len(body) != 0 {
				httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR",
					"Некорректный запрос", "Для назначения карточки требуется пустое тело запроса.")
				return
			}
		}
		analyst, _ := auth.PrincipalFromContext(r.Context())
		review, err := service.Claim(r.Context(), caseID, analyst.Subject)
		writeReviewMutationResult(w, r, review, err)
	}
}

func resolveReviewCaseHandler(service *services.ReviewDecisionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		caseID, ok := reviewMutationCaseID(w, r)
		if !ok {
			return
		}
		input, err := request.ParseReviewDecision(http.MaxBytesReader(w, r.Body, 1<<20))
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", err.Error())
			return
		}
		analyst, _ := auth.PrincipalFromContext(r.Context())
		review, err := service.Resolve(r.Context(), caseID, analyst.Subject, input.Decision, input.Comment)
		writeReviewMutationResult(w, r, review, err)
	}
}

func reviewMutationCaseID(w http.ResponseWriter, r *http.Request) (int32, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "caseId"), 10, 32)
	if err != nil || id < 1 {
		httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос",
			"caseId должен быть положительным целым числом типа int32.")
		return 0, false
	}
	return int32(id), true
}

func writeReviewMutationResult(w http.ResponseWriter, r *http.Request, review models.ReviewCase, err error) {
	switch {
	case errors.Is(err, models.ErrReviewCaseNotFound):
		httpserver.WriteProblem(w, http.StatusNotFound, "NOT_FOUND", "Карточка не найдена", "Запрошенная карточка недоступна.")
	case errors.Is(err, models.ErrReviewCaseAssigned):
		httpserver.WriteProblem(w, http.StatusConflict, "CASE_ALREADY_ASSIGNED", "Карточка назначена",
			"Карточка уже назначена другому аналитику.")
	case errors.Is(err, models.ErrReviewCaseClosed):
		httpserver.WriteProblem(w, http.StatusConflict, "CASE_CLOSED", "Карточка закрыта", "Карточка уже закрыта.")
	case err != nil:
		httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка",
			"Не удалось изменить карточку.")
	default:
		render.JSON(w, r, response.FromReviewCase(review))
	}
}
