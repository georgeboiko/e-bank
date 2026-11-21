package auth

import (
	"net/http"
	"slices"

	"e-bank/backend/internal/infrastructure/httpserver"
)

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				writeUnauthorized(w)
				return
			}
			if !slices.ContainsFunc(roles, principal.HasRole) {
				httpserver.WriteProblem(w, http.StatusForbidden, "FORBIDDEN",
					"Недостаточная роль", "У пользователя нет роли, необходимой для операции.")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func writeUnauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	httpserver.WriteProblem(w, http.StatusUnauthorized, "UNAUTHORIZED",
		"Требуется авторизация", "Отсутствует действительный access token.")
}
