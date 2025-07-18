package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"quickdocs/internal/auth"
)

type ctxKey string

const (
	userIDKey ctxKey = "userID"
	loginKey  ctxKey = "login"
)

type UserInfo struct {
	ID    int
	Login string
}

type AuthMiddleware struct {
	authService *auth.Service
}

func NewAuthMiddleware(authService *auth.Service) *AuthMiddleware {
	return &AuthMiddleware{authService: authService}
}

func (am *AuthMiddleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Неавторизованный: отсутствует заголовок Authorization header", http.StatusUnauthorized)
			return
		}

		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			http.Error(w, "Недопустимый формат токена", http.StatusUnauthorized)
			return
		}
		token := strings.TrimPrefix(authHeader, bearerPrefix)

		log.Println("Токен сгенерирован:", token)
		user, err := am.authService.ValidateToken(r.Context(), token)
		if err != nil {
			http.Error(w, "Полдьзователь не авторизован", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, user.ID)
		ctx = context.WithValue(ctx, loginKey, user.Login)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(ctx context.Context) (int, bool) {
	id, ok := ctx.Value(userIDKey).(int)
	return id, ok
}
