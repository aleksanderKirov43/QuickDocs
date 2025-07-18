package docs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"quickdocs/internal/middleware"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service    DocumentService
	fileFolder string
}

type DocumentService interface {
	CreateDocument(ctx context.Context, doc *Document) error
	GetDocument(ctx context.Context, id uuid.UUID) (*Document, error)
	DeleteDocument(ctx context.Context, id uuid.UUID) error
	ListDocumentsForUser(ctx context.Context, userID int, limit, offset int) ([]*Document, error)
	ListAll(ctx context.Context) ([]Document, error)
	GetFileMetadata(ctx context.Context, userID int, docID string) (*Document, error)
	ListFilesByUser(ctx context.Context, userID int) ([]Document, error)
}

func NewHandler(service DocumentService, fileFolder string) *Handler {
	return &Handler{service: service, fileFolder: fileFolder}
}

func (h *Handler) Upload(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Неавторизованный пользователь", http.StatusUnauthorized)
		return
	}

	err := r.ParseMultipartForm(32 << 20) // 32 MB max memory
	if err != nil {
		http.Error(w, "Не удалось разобрать multipart form: "+err.Error(), http.StatusBadRequest)
		return
	}

	metaJSON := r.FormValue("meta")
	if metaJSON == "" {
		http.Error(w, "Отсутствующее мета-поле", http.StatusBadRequest)
		return
	}

	var doc Document
	if err = json.Unmarshal([]byte(metaJSON), &doc); err != nil {
		http.Error(w, "Недопустимый мета-код JSON: "+err.Error(), http.StatusBadRequest)
		return
	}

	doc.OwnerID = userID
	if doc.ID == uuid.Nil {
		doc.ID = uuid.New()
	}

	file, header, err := r.FormFile("file")
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		http.Error(w, "Не удалось прочитать файл: "+err.Error(), http.StatusBadRequest)
		return
	}

	if file != nil {
		defer file.Close()

		// Проверим и сохраним файл локально
		filename := doc.ID.String() + "-" + filepath.Base(header.Filename)
		savedPath := filepath.Join(h.fileFolder, filename)

		outFile, err := os.Create(savedPath)
		if err != nil {
			http.Error(w, "Не удалось сохранить файл: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, file)
		if err != nil {
			http.Error(w, "Не удалось сохранить файл: "+err.Error(), http.StatusInternalServerError)
			return
		}

		doc.File = true
		doc.FilePath = savedPath
		doc.Name = header.Filename
		doc.Mime = header.Header.Get("Content-Type")
	} else {
		doc.File = false
		doc.FilePath = ""
	}

	if err := h.service.CreateDocument(r.Context(), &doc); err != nil {
		http.Error(w, "Не удалось создать документ: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": map[string]interface{}{
			"json": doc.JsonData,
			"file": doc.Name,
		},
	})
}

// Получаем документ по ID, если есть файл - отдаём его, иначе JSON
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Неверный ID документа", http.StatusBadRequest)
		return
	}

	_, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Неавторизованный", http.StatusUnauthorized)
		return
	}

	doc, err := h.service.GetDocument(r.Context(), id)
	if err != nil {
		http.Error(w, "Не удалось получить документ: "+err.Error(), http.StatusInternalServerError)
		return
	}
	if doc == nil {
		http.NotFound(w, r)
		return
	}

	// Проверка прав доступа (можно подключить опционально при необходимости)
	//if !doc.Public && doc.OwnerID != userID && !stringSliceContains(doc.Grant, getUserLoginFromContext(r.Context())) {
	//	http.Error(w, "Запрещенный", http.StatusForbidden)
	//	return
	//}
	if doc.File {
		doc.FilePath = filepath.Join(h.fileFolder, doc.ID.String()+"-"+doc.Name)
	}

	if doc.File && doc.FilePath != "" {
		w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(doc.FilePath))
		http.ServeFile(w, r, doc.FilePath)
		return
	}

	// Отдаём JSON из doc.JsonData
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"data": doc.JsonData,
	})
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		http.Error(w, "Недействительный ID документа", http.StatusBadRequest)
		return
	}

	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Неавторизованный", http.StatusUnauthorized)
		return
	}

	doc, err := h.service.GetDocument(r.Context(), id)
	if err != nil || doc == nil {
		http.Error(w, "Документ не найден", http.StatusNotFound)
		return
	}

	if doc.OwnerID != userID {
		http.Error(w, "Запрещенный", http.StatusForbidden)
		return
	}

	if err := h.service.DeleteDocument(r.Context(), id); err != nil {
		http.Error(w, "Не удалось удалить документ: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Удаляем файл, если есть
	if doc.File && doc.FilePath != "" {
		_ = os.Remove(doc.FilePath)
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"response": map[string]bool{
			idStr: true,
		},
	})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Неавторизованный", http.StatusUnauthorized)
		return
	}
	limit := 10
	offset := 0

	if v := r.URL.Query().Get("limit"); v != "" {
		if val, err := strconv.Atoi(v); err == nil && val > 0 {
			limit = val
		}
	}

	if v := r.URL.Query().Get("offset"); v != "" {
		if val, err := strconv.Atoi(v); err == nil && val >= 0 {
			offset = val
		}
	}

	docs, err := h.service.ListDocumentsForUser(r.Context(), userID, limit, offset)
	if err != nil {
		http.Error(w, "Не удалось составить список документов: "+err.Error(), http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, docs)
}

func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func (h *Handler) ListAll(w http.ResponseWriter, r *http.Request) {
	docs, err := h.service.ListAll(r.Context())
	if err != nil {
		http.Error(w, "Ошибка при получении всех документов: "+err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, docs)
}

func (h *Handler) Head(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Неавторизованный", http.StatusUnauthorized)
		return
	}

	fileID := chi.URLParam(r, "id")

	meta, err := h.service.GetFileMetadata(r.Context(), userID, fileID)
	if err != nil {
		http.Error(w, "Файл не найден", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+meta.Name)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", meta.Size))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HeadSessionCheck(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.GetUserID(r.Context())
	if !ok {
		http.Error(w, "Неавторизованный", http.StatusUnauthorized)
		return
	}

	// Просто проверка авторизации, можно добавить заголовки, если надо
	w.WriteHeader(http.StatusOK)
}
