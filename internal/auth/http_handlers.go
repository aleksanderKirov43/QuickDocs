package auth

import (
	"encoding/json"

	"net/http"
	passwords "quickdocs/pkg"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	Service *Service
}

//type LoginRequest struct {
//	Login string `json:"login"`
//	Pswd  string `json:"pswd"`
//}
//
//type LoginResponse struct {
//	Response *struct {
//		Token string `json:"token"`
//	} `json:"response,omitempty"`
//	Error *struct {
//		Code int    `json:"code"`
//		Text string `json:"text"`
//	} `json:"error,omitempty"`
//}

func NewHandler(s *Service) *Handler {
	return &Handler{Service: s}
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, 400, "Недействительный JSON")
		return
	}

	err := h.Service.RegisterUser(r.Context(), req.Login, req.Pswd)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}

	resp := RegisterResponse{
		Response: &RegisterData{Login: req.Login},
	}
	json.NewEncoder(w).Encode(resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeLoginError(w, 400, "Недействительный JSON")
		return
	}
	// Получаем ID и хэш пароля из БД
	_, hash, err := h.Service.userRepo.GetUserByLogin(r.Context(), req.Login)
	if err != nil {
		writeLoginError(w, 401, "Неверный логин или пароль")
		return
	}
	// Сравниваем пароль
	if !passwords.ComparePassword(hash, req.Pswd) {
		writeLoginError(w, 401, "Неверный логин или пароль")
		return
	}

	token, err := h.Service.GenerateToken(req.Login)
	if err != nil {
		writeLoginError(w, 500, "Ошибка генерации токена")
		return
	}
	resp := LoginResponse{
		Response: &LoginData{Token: token},
	}
	json.NewEncoder(w).Encode(resp)
}

func writeLoginError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(LoginResponse{
		Error: &ErrorResponse{Code: code, Text: msg},
	})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		writeLogoutError(w, 400, "Токен не передан")
		return
	}

	err := h.Service.Logout(r.Context(), token)
	if err != nil {
		writeLogoutError(w, 400, err.Error())
		return
	}

	resp := map[string]interface{}{
		"response": map[string]bool{
			token: true,
		},
	}
	json.NewEncoder(w).Encode(resp)
}

func writeLogoutError(w http.ResponseWriter, code int, msg string) {
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"error": map[string]interface{}{
			"code": code,
			"text": msg,
		},
	})
}
