package docs

import (
	"context"
	"net/http"
	"strconv"

	"quickdocs/internal/middleware"
	"quickdocs/internal/responses"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type DocumentService interface {
	// Создание документа из multipart запроса
	CreateDocumentFromUpload(ctx context.Context, userID int, r *http.Request, fileFolder string) (*Document, error)

	// Получение списка документов с фильтрами
	ListDocuments(ctx context.Context, userID int, filters ListFilters) ([]*Document, error)

	// Получение файла с проверкой прав
	GetFileBytes(ctx context.Context, userID int, id uuid.UUID) ([]byte, error)

	// Удаление документа с проверкой прав
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
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		responses.Error200(w, http.StatusUnauthorized, "Неавторизованный пользователь")
		return
	}

	doc, err := h.service.CreateDocumentFromUpload(r.Context(), userID, r, h.fileFolder)
	if err != nil {
		responses.Error200(w, http.StatusBadRequest, err.Error())
		return
	}

	responses.Created(w, map[string]interface{}{
		"json": doc.JsonData,
		"file": doc.Name,
	})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		responses.Error200(w, http.StatusBadRequest, "Неверный ID документа")
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		responses.Error200(w, http.StatusUnauthorized, "Неавторизованный")
		return
	}

	// Получаем файл через сервис (он сам проверит права и вернёт файл или ошибку)
	data, err := h.service.GetFileBytes(r.Context(), userID, id)
	if err != nil {
		responses.Error200(w, http.StatusBadRequest, err.Error())
		return
	}

	// Если это файл, отдаём его
	w.Header().Set("Content-Disposition", "attachment; filename="+idStr)
	w.Header().Set("Content-Type", "application/octet-stream")
	_, _ = w.Write(data)
}

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

func (h *Handler) Head(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.GetUserID(r.Context())
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}

	// HEAD запрос - просто проверяем авторизацию
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HeadSessionCheck(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}
	// Прогреваем кэш списка пользователя
	_, _ = h.service.ListDocuments(r.Context(), userID, ListFilters{Limit: 10, Offset: 0})
	w.WriteHeader(http.StatusOK)
}
