package middleware

import (
	"net/http"

	"quickdocs/config"
	"quickdocs/internal/responses"
)

func AdminToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Admin-Token")
		if token == "" {
			token = r.URL.Query().Get("admin_token")
		}
		if token == "" || token != config.Cfg.AdminToken {
			responses.Error200(w, http.StatusUnauthorized, "Недействительный токен администратора")
			return
		}
		next.ServeHTTP(w, r)
	})
}