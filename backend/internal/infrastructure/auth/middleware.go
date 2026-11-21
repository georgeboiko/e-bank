package auth

import (
	"net/http"

	"e-bank/backend/internal/infrastructure/httpserver"
)

func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			principal, ok := PrincipalFromContext(r.Context())
			if !ok {
				w.Header().Set("WWW-Authenticate", "Bearer")
				httpserver.WriteProblem(w, http.StatusUnauthorized, "UNAUTHORIZED",
					"Требуется авторизация", "Отсутствует действительный access token.")
				return
			}
			if !principal.HasRole(role) {
				httpserver.WriteProblem(w, http.StatusForbidden, "FORBIDDEN",
					"Недостаточная роль", "Для операции нужна роль "+role+".")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
