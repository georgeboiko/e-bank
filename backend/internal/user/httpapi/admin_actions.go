package httpapi

import (
	"errors"
	"net/http"

	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/user/dto/request"
	"e-bank/backend/internal/user/dto/response"
	"e-bank/backend/internal/user/keycloak"
	"e-bank/backend/internal/user/repositories"
	"e-bank/backend/internal/user/services"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
	"github.com/jackc/pgx/v5/pgxpool"
)

func registerAdminActionRoutes(router chi.Router, pool *pgxpool.Pool, directory *keycloak.Client) {
	service := services.NewAdminActionService(directory, repositories.NewAdminAuditRepository(pool))
	admin := router.With(auth.RequireRole("ADMIN"))
	admin.Patch("/admin/users/{userId}/status", setUserStatusHandler(service))
	admin.Put("/admin/users/{userId}/roles", setUserRolesHandler(service))
	admin.Delete("/admin/users/{userId}/sessions", terminateSessionsHandler(service))
}

func setUserStatusHandler(service *services.AdminActionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := userID(r)
		if err != nil {
			writeInvalidUserAction(w, err)
			return
		}
		enabled, err := request.ParseUserStatus(http.MaxBytesReader(w, r.Body, 4<<10))
		if err != nil {
			writeInvalidUserAction(w, err)
			return
		}
		principal, _ := auth.PrincipalFromContext(r.Context())
		user, err := service.SetStatus(r.Context(), principal.Subject, id, enabled)
		if err != nil {
			writeActionError(w, err)
			return
		}
		render.JSON(w, r, response.FromUser(user))
	}
}

func setUserRolesHandler(service *services.AdminActionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := userID(r)
		if err != nil {
			writeInvalidUserAction(w, err)
			return
		}
		roles, err := request.ParseUserRoles(http.MaxBytesReader(w, r.Body, 4<<10))
		if err != nil {
			writeInvalidUserAction(w, err)
			return
		}
		principal, _ := auth.PrincipalFromContext(r.Context())
		user, err := service.SetRoles(r.Context(), principal.Subject, id, roles)
		if err != nil {
			writeActionError(w, err)
			return
		}
		render.JSON(w, r, response.FromUser(user))
	}
}

func terminateSessionsHandler(service *services.AdminActionService) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := userID(r)
		if err != nil {
			writeInvalidUserAction(w, err)
			return
		}
		principal, _ := auth.PrincipalFromContext(r.Context())
		if err := service.TerminateSessions(r.Context(), principal.Subject, id); err != nil {
			writeUserError(w, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}
}

func writeInvalidUserAction(w http.ResponseWriter, err error) {
	httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", err.Error())
}

func writeActionError(w http.ResponseWriter, err error) {
	if errors.Is(err, keycloak.ErrUserConflict) {
		httpserver.WriteProblem(w, http.StatusConflict, "USER_CONFLICT", "Конфликт состояния пользователя",
			"Не удалось применить изменение в Keycloak.")
		return
	}
	writeUserError(w, err)
}
