package auth

import (
	"encoding/json"
	"github.com/go-chi/chi/v5"
	"net/http"
	"quickdocs/internal/responses"
)

type Handler struct {
	auth AuthService
}

func NewHandler(s AuthService) *Handler {
	return &Handler{auth: s}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responses.Fail(w, http.StatusBadRequest, http.StatusBadRequest, "Недействительный JSON")
		return
	}

	// Проверяем админ-токен согласно ТЗ (на уровне middleware должен быть)
	//if err := h.auth.CheckAdminToken(r.Context(), req.Token); err != nil {
	//	responses.Fail(w, http.StatusUnauthorized, http.StatusUnauthorized, err.Error())
	//	return
	//}

	err := h.auth.RegisterUser(r.Context(), req.Login, req.Pswd)
	if err != nil {
		responses.Fail(w, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}

	responses.Ack(w, &RegisterData{Login: req.Login})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	ct := r.Header.Get("Content-Type")
	if ct == "application/x-www-form-urlencoded" || ct == "application/x-www-form-urlencoded; charset=UTF-8" {
		if err := r.ParseForm(); err == nil {
			req.Login = r.FormValue("login")
			req.Pswd = r.FormValue("pswd")
		} else {
			responses.Fail(w, http.StatusBadRequest, http.StatusBadRequest, "Некорректная форма")
			return
		}
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			responses.Fail(w, http.StatusBadRequest, http.StatusBadRequest, "Недействительный JSON")
			return
		}
	}
	//// Получаем ID и хэш пароля из БД (Перенести логику в сервис)
	//_, hash, err := h.auth.GetUserByLogin(r.Context(), req.Login)
	//if err != nil {
	//	responses.Fail(w, http.StatusUnauthorized, http.StatusUnauthorized, "Неверный логин или пароль")
	//	return
	//}
	//// Сравниваем пароль
	//if !passwords.ComparePassword(hash, req.Pswd) {
	//	responses.Fail(w, http.StatusUnauthorized, http.StatusUnauthorized, "Неверный логин или пароль")
	//	return
	//}

	// (Перенести в сервис)
	token, err := h.auth.GenerateToken(req.Login)
	if err != nil {
		responses.Fail(w, http.StatusInternalServerError, http.StatusInternalServerError, "Ошибка генерации токена")
		return
	}
	responses.Ack(w, &LoginData{Token: token})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		responses.Fail(w, http.StatusBadRequest, http.StatusBadRequest, "Токен не передан")
		return
	}

	err := h.auth.Logout(r.Context(), token)
	if err != nil {
		responses.Fail(w, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}

	responses.Ack(w, map[string]bool{token: true})
}
