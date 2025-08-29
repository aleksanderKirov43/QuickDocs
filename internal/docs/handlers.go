package docs

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"io"
	"net/http"
	"strconv"

	"quickdocs/internal/middleware"
	"quickdocs/internal/responses"
)

type DocumentService interface {
	CreateDocument(ctx context.Context, userID int, file io.Reader, fileName, meta string) (*Document, error)
	ListDocuments(ctx context.Context, userID int, filters ListFilters) ([]*Document, error)
	GetDocument(ctx context.Context, userID int, id uuid.UUID) (*DocumentWithBytes, error)
	DeleteDocumentForUser(ctx context.Context, userID int, id uuid.UUID) error
}

type Handler struct {
	service    DocumentService
	fileFolder string
}

func NewHandler(service DocumentService, fileFolder string) *Handler {
	return &Handler{service: service, fileFolder: fileFolder}
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		responses.Error200(w, http.StatusUnauthorized, "Неавторизованный пользователь")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		responses.Error200(w, http.StatusBadRequest, "Файл не предоставлен")
		return
	}
	defer file.Close()

	meta := r.FormValue("meta")

	doc, err := h.service.CreateDocument(ctx, userID, file, header.Filename, meta)
	if err != nil {
		responses.Error200(w, http.StatusInternalServerError, "Не удалось создать документ: "+err.Error())
		return
	}

	responses.Created(w, map[string]interface{}{
		"data": map[string]interface{}{
			"file": doc.Name,
			"json": doc.JsonData,
		},
	})
}

// Get только вызывает сервис и отдаёт байты
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		responses.Error200(w, http.StatusBadRequest, "Неверный ID документа")
		return
	}

	userID, ok := middleware.GetUserID(ctx)
	if !ok {
		responses.Error200(w, http.StatusUnauthorized, "Неавторизованный")
		return
	}

	docWithBytes, err := h.service.GetDocument(ctx, userID, id)
	if err != nil {
		responses.Error200(w, http.StatusNotFound, err.Error())
		return
	}

	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", docWithBytes.Name))
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(docWithBytes.Data)
}

// Delete вызывает сервис
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		responses.Error200(w, http.StatusBadRequest, "Недействительный ID документа")
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		responses.Error200(w, http.StatusUnauthorized, "Неавторизованный")
		return
	}

	if err := h.service.DeleteDocumentForUser(r.Context(), userID, id); err != nil {
		responses.Error200(w, http.StatusBadRequest, err.Error())
		return
	}

	responses.Ack(w, map[string]bool{idStr: true})
}

// List только собирает фильтры и отдаёт сервису
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		responses.Error200(w, http.StatusUnauthorized, "Неавторизованный")
		return
	}

	filters := ListFilters{
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
		responses.Error200(w, http.StatusInternalServerError, "Не удалось составить список документов: "+err.Error())
		return
	}

	responses.OK(w, map[string]interface{}{"docs": docs})
}

// Прогрев кэша, только вызывает сервис
func (h *Handler) HeadSessionCheck(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}
	_, _ = h.service.ListDocuments(r.Context(), userID, ListFilters{Limit: 10, Offset: 0})
	w.WriteHeader(http.StatusOK)
}
