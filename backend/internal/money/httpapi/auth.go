package httpapi

import (
	"net/http"

	"e-bank/backend/internal/infrastructure/auth"
)

func requireClient(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		principal, ok := auth.PrincipalFromContext(r.Context())
		if !ok {
			w.Header().Set("WWW-Authenticate", "Bearer")
			writeProblem(w, http.StatusUnauthorized, "UNAUTHORIZED",
				"Требуется авторизация", "Отсутствует действительный access token.")
			return
		}
		if !principal.HasRole("CLIENT") {
			writeProblem(w, http.StatusForbidden, "FORBIDDEN",
				"Недостаточная роль", "Для операции нужна роль CLIENT.")
			return
		}

		next.ServeHTTP(w, r)
	})
}
