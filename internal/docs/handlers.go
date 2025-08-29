package docs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"quickdocs/internal/middleware"
	"quickdocs/internal/responses"
)

type DocumentService interface {
	CreateDocument(ctx context.Context, doc *Document) error
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

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		responses.Error200(w, http.StatusBadRequest, "Неверные данные формы")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		responses.Error200(w, http.StatusBadRequest, "Файл не предоставлен")
		return
	}
	defer file.Close()

	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		responses.Error200(w, http.StatusInternalServerError, "Невозможно создать директорию")
		return
	}

	fileID := uuid.New()
	filePath := filepath.Join(uploadDir, fileID.String()+"_"+header.Filename)

	dst, err := os.Create(filePath)
	if err != nil {
		responses.Error200(w, http.StatusInternalServerError, "Невозможно сохранить файл")
		return
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		responses.Error200(w, http.StatusInternalServerError, "Ошибка записи файла")
		return
	}

	// Парсим meta JSON
	var jsonData *json.RawMessage
	public := false
	if meta := r.FormValue("meta"); meta != "" {
		var metaMap map[string]interface{}
		if err := json.Unmarshal([]byte(meta), &metaMap); err != nil {
			responses.Error200(w, http.StatusBadRequest, "Неверный формат meta JSON")
			return
		}
		jm := json.RawMessage(meta)
		jsonData = &jm

		if pub, ok := metaMap["public"].(bool); ok {
			public = pub
		}
	}

	doc := &Document{
		ID:       fileID,
		OwnerID:  userID,
		Name:     header.Filename,
		File:     true,
		Public:   public,
		JsonData: jsonData,
	}

	if err := h.service.CreateDocument(ctx, doc); err != nil {
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
