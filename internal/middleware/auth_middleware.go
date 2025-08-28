package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"

	"quickdocs/internal/auth"
	"quickdocs/internal/responses"
)

type ctxKey string

const (
	userIDKey ctxKey = "userID"
	loginKey  ctxKey = "login"
	tokenKey  ctxKey = "token"
)

type UserInfo struct {
	ID    int
	Login string
}

type AuthMiddleware struct {
	authService auth.AuthService
}

func NewAuthMiddleware(authService auth.AuthService) *AuthMiddleware {
	return &AuthMiddleware{authService: authService}
}

func (am *AuthMiddleware) Auth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// request id
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			reqID = generateRequestID()
		}
		r = r.WithContext(context.WithValue(r.Context(), ctxKey("reqID"), reqID))
		authHeader := r.Header.Get("Authorization")
		token := ""

		// ТЗ допускает токен как параметр token
		if q := r.URL.Query().Get("token"); q != "" {
			token = q
		}

		if token == "" {
			if authHeader == "" {
				responses.Error200(w, http.StatusUnauthorized, "Неавторизованный: отсутствует токен")
				return
			}

			const bearerPrefix = "Bearer "
			if !strings.HasPrefix(authHeader, bearerPrefix) {
				responses.Error200(w, http.StatusUnauthorized, "Недопустимый формат токена")
				return
			}
			token = strings.TrimPrefix(authHeader, bearerPrefix)
		}

		log.Println("Токен сгенерирован:", token)
		user, err := am.authService.ValidateToken(r.Context(), token)
		if err != nil {
			responses.Error200(w, http.StatusUnauthorized, "Пользователь не авторизован")
			return
		}

		ctx := context.WithValue(r.Context(), userIDKey, user.ID)
		ctx = context.WithValue(ctx, loginKey, user.Login)
		ctx = context.WithValue(ctx, tokenKey, token)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(ctx context.Context) (int, bool) {
	id, ok := ctx.Value(userIDKey).(int)
	return id, ok
}

func GetLogin(ctx context.Context) (string, bool) {
	login, ok := ctx.Value(loginKey).(string)
	return login, ok
}

func GetToken(ctx context.Context) (string, bool) {
	t, ok := ctx.Value(tokenKey).(string)
	return t, ok
}
