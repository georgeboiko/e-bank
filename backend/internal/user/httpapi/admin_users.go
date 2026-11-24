package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"e-bank/backend/internal/infrastructure/auth"
	"e-bank/backend/internal/infrastructure/httpserver"
	"e-bank/backend/internal/user/dto/request"
	"e-bank/backend/internal/user/dto/response"
	"e-bank/backend/internal/user/keycloak"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/render"
)

func registerAdminReadRoutes(router chi.Router, directory *keycloak.Client) {
	admin := router.With(auth.RequireRole("ADMIN"))
	admin.Get("/admin/users", listUsersHandler(directory))
	admin.Get("/admin/users/{userId}", getUserHandler(directory))
}

func listUsersHandler(directory *keycloak.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		query, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			writeUserValidationError(w, "Проверьте параметры запроса.")
			return
		}
		filter, err := request.ParseUserFilter(query)
		if err != nil {
			writeUserValidationError(w, err.Error())
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
		defer cancel()
		users, hasMore, err := directory.ListUsers(ctx, filter)
		if err != nil {
			writeUserError(w, err)
			return
		}
		render.JSON(w, r, response.ToUserPage(users, filter, hasMore))
	}
}

func getUserHandler(directory *keycloak.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := userID(r)
		if err != nil {
			writeUserValidationError(w, err.Error())
			return
		}

		user, err := directory.GetUser(r.Context(), id)
		if err != nil {
			writeUserError(w, err)
			return
		}
		render.JSON(w, r, response.FromUser(user))
	}
}

func userID(r *http.Request) (string, error) {
	id := chi.URLParam(r, "userId")
	if r.URL.RawPath != "" {
		decoded, err := url.PathUnescape(id)
		if err != nil {
			return "", errors.New("некорректный userId")
		}
		id = decoded
	}
	return request.ParseUserID(id)
}

func writeUserValidationError(w http.ResponseWriter, detail string) {
	httpserver.WriteProblem(w, http.StatusBadRequest, "VALIDATION_ERROR", "Некорректный запрос", detail)
}

func writeUserError(w http.ResponseWriter, err error) {
	if errors.Is(err, keycloak.ErrUserNotFound) {
		httpserver.WriteProblem(w, http.StatusNotFound, "NOT_FOUND", "Пользователь не найден", "Запрошенный пользователь недоступен.")
		return
	}
	httpserver.WriteProblem(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Внутренняя ошибка", "Не удалось получить данные Keycloak.")
}
