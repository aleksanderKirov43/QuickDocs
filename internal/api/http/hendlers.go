package http

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"quickdocs/app"
	"strconv"

	"quickdocs/internal/models"
	"quickdocs/internal/services"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type DocumentService interface {
	CreateDocument(ctx context.Context, userID int, file io.Reader, fileName, meta string) (*models.Document, error)
	ListDocuments(ctx context.Context, userID int, filters models.ListFilters) ([]*models.Document, error)
	GetDocument(ctx context.Context, userID int, id uuid.UUID) (*models.DocumentWithBytes, error)
	DeleteDocumentForUser(ctx context.Context, userID int, id uuid.UUID) error
}

type DocsHandler struct {
	service    DocumentService
	fileFolder string
}

type AuthHandler struct {
	Auth services.AuthService
}

func NewHandlerAuth(s services.AuthService) *AuthHandler {
	return &AuthHandler{Auth: s}
}

func NewHandlerDocs(service DocumentService, fileFolder string) *DocsHandler {
	return &DocsHandler{service: service, fileFolder: fileFolder}
}

func (h *DocsHandler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := app.GetUserID(ctx)
	if !ok {
		Error200(w, http.StatusUnauthorized, "Неавторизованный пользователь")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		Error200(w, http.StatusBadRequest, "Файл не предоставлен")
		return
	}
	defer file.Close()

	meta := r.FormValue("meta")

	doc, err := h.service.CreateDocument(ctx, userID, file, header.Filename, meta)
	if err != nil {
		Error200(w, http.StatusInternalServerError, "Не удалось создать документ: "+err.Error())
		return
	}

	Created(w, map[string]interface{}{
		"data": map[string]interface{}{
			"file": doc.Name,
			"json": doc.JsonData,
		},
	})
}

// Вызывает сервис и отдаёт байты
func (h *DocsHandler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error200(w, http.StatusBadRequest, "Неверный ID документа")
		return
	}

	userID, ok := app.GetUserID(ctx)
	if !ok {
		Error200(w, http.StatusUnauthorized, "Неавторизованный")
		return
	}

	docWithBytes, err := h.service.GetDocument(ctx, userID, id)
	if err != nil {
		Error200(w, http.StatusNotFound, err.Error())
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", docWithBytes.Name))
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(docWithBytes.Data)
}

// Delete вызывает сервис
func (h *DocsHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		Error200(w, http.StatusBadRequest, "Недействительный ID документа")
		return
	}

	userID, ok := app.GetUserID(r.Context())
	if !ok {
		Error200(w, http.StatusUnauthorized, "Неавторизованный")
		return
	}

	if err := h.service.DeleteDocumentForUser(r.Context(), userID, id); err != nil {
		Error200(w, http.StatusBadRequest, err.Error())
		return
	}

	Ack(w, map[string]bool{idStr: true})
}

// List только собирает фильтры и отдаёт сервису
func (h *DocsHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := app.GetUserID(r.Context())
	if !ok {
		Error200(w, http.StatusUnauthorized, "Неавторизованный")
		return
	}

	filters := models.ListFilters{
		Limit:  10,
		Offset: 0,
		Key:    r.URL.Query().Get("key"),
		Value:  r.URL.Query().Get("value"),
		SortBy: r.URL.Query().Get("sort"),
		Order:  r.URL.Query().Get("order"),
		Login:  r.URL.Query().Get("login"),
	}

	if v := r.URL.Query().Get("limit"); v != "" {
		if val, err := strconv.Atoi(v); err == nil && val > 0 {
			filters.Limit = val
		}
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		if val, err := strconv.Atoi(v); err == nil && val >= 0 {
			filters.Offset = val
		}
	}

	docs, err := h.service.ListDocuments(r.Context(), userID, filters)
	if err != nil {
		Error200(w, http.StatusInternalServerError, "Не удалось составить список документов: "+err.Error())
		return
	}

	OK(w, map[string]interface{}{"docs": docs})
}

// Прогрев кэша, только вызывает сервис
func (h *DocsHandler) HeadSessionCheck(w http.ResponseWriter, r *http.Request) {
	userID, ok := app.GetUserID(r.Context())
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}
	_, _ = h.service.ListDocuments(r.Context(), userID, models.ListFilters{Limit: 10, Offset: 0})
	w.WriteHeader(http.StatusOK)
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		Fail(w, http.StatusBadRequest, http.StatusBadRequest, "Недействительный JSON")
	}

	err := h.Auth.RegisterUser(r.Context(), req.Login, req.Pswd)
	if err != nil {
		Fail(w, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}

	Ack(w, &models.RegisterData{Login: req.Login})
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	ct := r.Header.Get("Content-Type")
	if ct == "application/x-www-form-urlencoded" || ct == "application/x-www-form-urlencoded; charset=UTF-8" {
		if err := r.ParseForm(); err == nil {
			req.Login = r.FormValue("login")
			req.Pswd = r.FormValue("pswd")
		} else {
			Fail(w, http.StatusBadRequest, http.StatusBadRequest, "Некорректная форма")
			return
		}
	} else {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			Fail(w, http.StatusBadRequest, http.StatusBadRequest, "Недействительный JSON")
			return
		}
	}

	token, err := h.Auth.Login(r.Context(), req.Login, req.Pswd)
	if err != nil {
		Fail(w, http.StatusInternalServerError, http.StatusInternalServerError, err.Error())
		return
	}
	Ack(w, &models.LoginData{Token: token})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		Fail(w, http.StatusBadRequest, http.StatusBadRequest, "Токен не передан")
		return
	}

	err := h.Auth.Logout(r.Context(), token)
	if err != nil {
		Fail(w, http.StatusBadRequest, http.StatusBadRequest, err.Error())
		return
	}

	Ack(w, map[string]bool{token: true})
}
