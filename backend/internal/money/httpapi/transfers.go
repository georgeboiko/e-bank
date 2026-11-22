package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"net/netip"
	"strconv"
	"time"

	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/money/dto/request"
	"e-bank/backend/internal/money/dto/response"
	"e-bank/backend/internal/money/models"
	"e-bank/backend/internal/money/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func createTransferHandler(transferService *services.TransferService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		input, err := request.ParseCreateTransfer(
			http.MaxBytesReader(w, r.Body, 1<<20), r.Header.Values("Idempotency-Key"),
		)
		if err != nil {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", err.Error())
			return
		}

		principal, _ := auth.PrincipalFromContext(r.Context())
		input.ActorID = principal.Subject
		input.IPAddress = transferClientIP(r)

		transfer, created, err := transferService.CreateTransfer(r.Context(), input)
		if err != nil {
			writeCreateTransferError(w, err)
			return
		}
		if created {
			w.Header().Set("Location", fmt.Sprintf("/api/money/v1/transfers/%d", transfer.ID))
			render.Status(r, http.StatusAccepted)
		}
		render.JSON(w, r, toTransferResponse(transfer))
	}
}

func getTransferHandler(transferService *services.TransferService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(chi.URLParam(r, "transferId"), 10, 32)
		if err != nil || id < 1 {
			httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос",
				"transferId должен быть положительным целым числом типа int32.")
			return
		}

		principal, _ := auth.PrincipalFromContext(r.Context())
		transfer, err := transferService.GetTransfer(r.Context(), principal.Subject, principal.HasRole("ADMIN"), int32(id))
		switch {
		case errors.Is(err, models.ErrTransferNotFound):
			httpserver.WriteProblem(w, http.StatusNotFound, "NOT_FOUND", "Перевод не найден", "Запрошенный перевод недоступен.")
		case err != nil:
			httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
				"Внутренняя ошибка", "Не удалось получить перевод.")
		default:
			render.JSON(w, r, toTransferResponse(transfer))
		}
	}
}

func transferClientIP(r *http.Request) netip.Addr {
	peer, err := netip.ParseAddrPort(r.RemoteAddr)
	if err != nil {
		return netip.Addr{}
	}
	ip := peer.Addr().Unmap()
	if ip.IsLoopback() {
		if forwarded, err := netip.ParseAddr(r.Header.Get("X-Real-IP")); err == nil {
			ip = forwarded.Unmap()
		}
	}
	return ip.WithZone("")
}

func writeCreateTransferError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, models.ErrInvalidTransfer):
		httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR",
			"Некорректный запрос", "Проверьте счета и сумму перевода.")
	case errors.Is(err, models.ErrAccountNotFound):
		httpserver.WriteProblem(w, http.StatusNotFound, "NOT_FOUND",
			"Счёт не найден", "Один из счетов перевода недоступен.")
	case errors.Is(err, models.ErrIdempotencyConflict):
		httpserver.WriteProblem(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT",
			"Ключ уже использован", "С этим ключом ранее отправили другой запрос.")
	case errors.Is(err, models.ErrInactiveAccount):
		httpserver.WriteProblem(w, http.StatusConflict, "ACCOUNT_NOT_ACTIVE",
			"Счёт неактивен", "Оба счёта должны быть активны.")
	case errors.Is(err, models.ErrInsufficientFunds):
		httpserver.WriteProblem(w, http.StatusConflict, "INSUFFICIENT_FUNDS",
			"Недостаточно средств", "Сумма превышает доступный остаток.")
	case errors.Is(err, models.ErrTransferLimit):
		httpserver.WriteProblem(w, http.StatusConflict, "TRANSFER_LIMIT_EXCEEDED",
			"Превышен лимит", "Сумма превышает остаток дневного или месячного лимита переводов.")
	default:
		httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR",
			"Внутренняя ошибка", "Не удалось создать перевод.")
	}
}

func toTransferResponse(transfer models.Transfer) response.Transfer {
	var finishedAt *time.Time
	if transfer.FinishedAt != nil {
		utc := transfer.FinishedAt.UTC()
		finishedAt = &utc
	}
	return response.Transfer{
		ID:            transfer.ID,
		Type:          transfer.Type,
		FromAccountID: transfer.FromAccountID,
		ToAccountID:   transfer.ToAccountID,
		Amount:        transfer.Amount.StringFixed(2),
		State:         transfer.State,
		RejectCause:   transfer.RejectCause,
		CreatedAt:     transfer.CreatedAt.UTC(),
		UpdatedAt:     transfer.UpdatedAt.UTC(),
		FinishedAt:    finishedAt,
	}
}
