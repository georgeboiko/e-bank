package httpapi

import (
	"errors"
	"fmt"
	"io"
	"net/http"
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

func createTopupHandler(service *services.TopupService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		input, err := request.ParseCreateTopup(http.MaxBytesReader(w, r.Body, 1<<20), r.Header.Values("Idempotency-Key"))
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", err.Error())
			return
		}
		principal, _ := auth.PrincipalFromContext(r.Context())
		input.ActorID = principal.Subject
		topup, created, err := service.Create(r.Context(), input)
		if err != nil {
			writeTopupError(w, err)
			return
		}
		if created {
			w.Header().Set("Location", fmt.Sprintf("/api/money/v1/topups/%d", topup.ID))
			render.Status(r, http.StatusCreated)
		}
		render.JSON(w, r, response.FromTopup(topup))
	}
}

func getTopupHandler(service *services.TopupService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := topupPathID(w, r)
		if !ok {
			return
		}
		principal, _ := auth.PrincipalFromContext(r.Context())
		topup, err := service.Get(r.Context(), principal.Subject, principal.HasRole("ADMIN"), id)
		if err != nil {
			writeTopupError(w, err)
			return
		}
		render.JSON(w, r, response.FromTopup(topup))
	}
}

func refreshTopupHandler(service *services.TopupService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, ok := topupPathID(w, r)
		if !ok {
			return
		}
		if r.Body != nil && r.Body != http.NoBody {
			body, err := io.ReadAll(io.LimitReader(r.Body, 1))
			if err != nil || len(body) != 0 {
				httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR",
					"Некорректный запрос", "Для проверки оплаты требуется пустое тело запроса.")
				return
			}
		}
		principal, _ := auth.PrincipalFromContext(r.Context())
		topup, err := service.Refresh(r.Context(), principal.Subject, id)
		if err != nil {
			writeTopupError(w, err)
			return
		}
		render.JSON(w, r, response.FromTopup(topup))
	}
}

func topupPathID(w http.ResponseWriter, r *http.Request) (int32, bool) {
	id, err := strconv.ParseInt(chi.URLParam(r, "topupId"), 10, 32)
	if err != nil || id < 1 {
		httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос",
			"topupId должен быть положительным целым числом типа int32.")
		return 0, false
	}
	return int32(id), true
}

func writeTopupError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrInvalidTransfer):
		httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", "Недопустимые параметры пополнения.")
	case errors.Is(err, models.ErrAccountNotFound):
		httpserver.WriteProblem(w, http.StatusNotFound, "NOT_FOUND", "Счёт не найден", "Счёт для пополнения недоступен.")
	case errors.Is(err, models.ErrTopupNotFound):
		httpserver.WriteProblem(w, http.StatusNotFound, "NOT_FOUND", "Пополнение не найдено", "Запрошенное пополнение недоступно.")
	case errors.Is(err, models.ErrIdempotencyConflict):
		httpserver.WriteProblem(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT",
			"Конфликт ключа запроса", "Ключ уже использован для другого запроса.")
	case errors.Is(err, models.ErrInactiveAccount):
		httpserver.WriteProblem(w, http.StatusConflict, "ACCOUNT_INACTIVE", "Счёт не активен", "Для операции нужны активные счета.")
	case errors.Is(err, models.ErrTopupPaymentMismatch):
		httpserver.WriteProblem(w, http.StatusConflict, "PAYMENT_MISMATCH", "Платёж не совпадает", "Параметры платежа не соответствуют пополнению.")
	case errors.Is(err, models.ErrTopupNoPayment):
		httpserver.WriteProblem(w, http.StatusConflict, "PAYMENT_NOT_REGISTERED", "Платёж не привязан", "Сначала подготовьте внешний платёж.")
	case errors.Is(err, models.ErrTopupInvalidState):
		httpserver.WriteProblem(w, http.StatusConflict, "INVALID_TOPUP_STATE", "Недопустимое состояние", "Текущее состояние не принимает этот результат оплаты.")
	case errors.Is(err, models.ErrTopupRetryExpired):
		httpserver.WriteProblem(w, http.StatusConflict, "PAYMENT_RETRY_EXPIRED", "Срок повтора истёк", "Пополнение требует ручной сверки.")
	default:
		httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка", "Не удалось обработать пополнение.")
	}
}
