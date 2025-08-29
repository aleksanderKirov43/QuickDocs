package pkg

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"quickdocs/config"
	http2 "quickdocs/internal/api/http"
)

// GenerateRequestID — генерация уникального ID для запроса
func GenerateRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// AdminToken — middleware для проверки админского токена
func AdminToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("X-Admin-Token")
		if token == "" {
			token = r.URL.Query().Get("admin_token")
		}
		if token == "" || token != config.Cfg.AdminToken {
			http2.Error200(w, http.StatusUnauthorized, "Недействительный токен администратора")
			return
		}
		next.ServeHTTP(w, r)
	})
}
