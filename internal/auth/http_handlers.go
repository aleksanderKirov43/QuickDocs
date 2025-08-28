package auth

import (
	"encoding/json"
	"net/http"

	"quickdocs/internal/responses"

	"github.com/go-chi/chi/v5"
)

func NewHandler(s AuthService) *Handler {
	return &Handler{auth: s}
}

func (h *Handler) Service() AuthService {
	return h.auth
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		responses.Fail(w, http.StatusBadRequest, http.StatusBadRequest, "Недействительный JSON")
		return
	}

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

	token, err := h.auth.Login(r.Context(), req.Login, req.Pswd)
	if err != nil {
		responses.Fail(w, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
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
