package docs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"quickdocs/internal/middleware"
	"quickdocs/internal/responses"

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
	GetDocumentForUser(ctx context.Context, userID int, id uuid.UUID) (*Document, error)
	DeleteDocument(ctx context.Context, id uuid.UUID) error
	ListDocumentsForUser(ctx context.Context, userID int, limit, offset int) ([]*Document, error)
	ListAll(ctx context.Context) ([]Document, error)
	GetFileMetadata(ctx context.Context, userID int, docID string) (*Document, error)
	ListFilesByUser(ctx context.Context, userID int) ([]Document, error)
	ListDocumentsFiltered(ctx context.Context, userID int, key, value string, limit, offset int, sortBy, order string) ([]*Document, error)
	ListPublicByLogin(ctx context.Context, login string, key, value string, limit, offset int, sortBy, order string) ([]*Document, error)
	GetFileBytesCached(ctx context.Context, userID int, id uuid.UUID, path string, mime string) ([]byte, error)
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

	err := r.ParseMultipartForm(32 << 20) // 32 MB max memory
	if err != nil {
		responses.Error200(w, http.StatusBadRequest, "Не удалось разобрать multipart form: "+err.Error())
		return
	}

	metaJSON := r.FormValue("meta")
	if metaJSON == "" {
		responses.Error200(w, http.StatusBadRequest, "Отсутствующее мета-поле")
		return
	}

	var doc Document
	if err = json.Unmarshal([]byte(metaJSON), &doc); err != nil {
		responses.Error200(w, http.StatusBadRequest, "Недопустимый мета-код JSON: "+err.Error())
		return
	}

	// опциональное поле json
	if jsonStr := r.FormValue("json"); jsonStr != "" {
		b := json.RawMessage(jsonStr)
		doc.JsonData = &b
	}

	doc.OwnerID = userID
	if doc.ID == uuid.Nil {
		doc.ID = uuid.New()
	}

	file, header, err := r.FormFile("file")
	if err != nil && !errors.Is(err, http.ErrMissingFile) {
		responses.Error200(w, http.StatusBadRequest, "Не удалось прочитать файл: "+err.Error())
		return
	}

	if file != nil {
		defer file.Close()

		// Проверим и сохраним файл локально
		filename := doc.ID.String() + "-" + filepath.Base(header.Filename)
		savedPath := filepath.Join(h.fileFolder, filename)

		outFile, err := os.Create(savedPath)
		if err != nil {
			responses.Error200(w, http.StatusInternalServerError, "Не удалось сохранить файл: "+err.Error())
			return
		}
		defer outFile.Close()

		_, err = io.Copy(outFile, file)
		if err != nil {
			responses.Error200(w, http.StatusInternalServerError, "Не удалось сохранить файл: "+err.Error())
			return
		}

		// Валидация MIME: по расширению и заголовку
		ext := filepath.Ext(header.Filename)
		guessed := mime.TypeByExtension(ext)
		contentType := header.Header.Get("Content-Type")
		if contentType == "" {
			contentType = guessed
		}
		doc.File = true
		doc.FilePath = savedPath
		doc.Name = header.Filename
		doc.Mime = contentType
	} else {
		doc.File = false
		doc.FilePath = ""
	}

	if err := h.service.CreateDocument(r.Context(), &doc); err != nil {
		responses.Error200(w, http.StatusInternalServerError, "Не удалось создать документ: "+err.Error())
		return
	}

	responses.Created(w, map[string]interface{}{
		"json": doc.JsonData,
		"file": doc.Name,
	})
}

// Получаем документ по ID, если есть файл - отдаём его, иначе JSON
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

	doc, err := h.service.GetDocumentForUser(r.Context(), userID, id)
	if err != nil {
		responses.Error200(w, http.StatusInternalServerError, "Не удалось получить документ: "+err.Error())
		return
	}
	if doc == nil {
		responses.Error200(w, http.StatusBadRequest, "Документ не найден")
		return
	}

	if doc.File {
		doc.FilePath = filepath.Join(h.fileFolder, doc.ID.String()+"-"+doc.Name)
	}

	if doc.File && doc.FilePath != "" {
		w.Header().Set("Content-Disposition", "attachment; filename="+filepath.Base(doc.FilePath))
		w.Header().Set("Content-Type", doc.Mime)

		// Попытка отдать из кэша байтов
		if data, err := h.service.GetFileBytesCached(r.Context(), userID, id, doc.FilePath, doc.Mime); err == nil && len(data) > 0 {
			_, _ = w.Write(data)
			return
		}

		// Fallback уже выполнится в сервисе; если там ошибка — вернём 200+error
		responses.Error200(w, http.StatusInternalServerError, "Не удалось отдать файл")
		return
	}

	// Отдаём JSON из doc.JsonData
	responses.OK(w, doc.JsonData)
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

	doc, err := h.service.GetDocument(r.Context(), id)
	if err != nil || doc == nil {
		responses.Error200(w, http.StatusBadRequest, "Документ не найден")
		return
	}

	if doc.OwnerID != userID {
		responses.Error200(w, http.StatusForbidden, "Запрещенный")
		return
	}

	if err := h.service.DeleteDocument(r.Context(), id); err != nil {
		responses.Error200(w, http.StatusInternalServerError, "Не удалось удалить документ: "+err.Error())
		return
	}

	// Удаляем файл, если есть
	if doc.File && doc.FilePath != "" {
		_ = os.Remove(doc.FilePath)
	}

	responses.Ack(w, map[string]bool{idStr: true})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	_, ok := middleware.GetUserID(r.Context())
	if !ok {
		responses.Error200(w, http.StatusUnauthorized, "Неавторизованный")
		return
	}

	loginParam := r.URL.Query().Get("login")

	limit := 10
	offset := 0
	key := r.URL.Query().Get("key")
	value := r.URL.Query().Get("value")
	sortBy := r.URL.Query().Get("sort")
	order := r.URL.Query().Get("order")

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

	var (
		docs []*Document
		err  error
	)

	if loginParam != "" {
		docs, err = h.service.ListPublicByLogin(r.Context(), loginParam, key, value, limit, offset, sortBy, order)
	} else {
		userID, _ := middleware.GetUserID(r.Context())
		if key != "" || value != "" || sortBy != "" || order != "" {
			docs, err = h.service.ListDocumentsFiltered(r.Context(), userID, key, value, limit, offset, sortBy, order)
		} else {
			docs, err = h.service.ListDocumentsForUser(r.Context(), userID, limit, offset)
		}
	}
	if err != nil {
		responses.Error200(w, http.StatusInternalServerError, "Не удалось составить список документов: "+err.Error())
		return
	}

	responses.OK(w, map[string]interface{}{"docs": docs})
}

func (h *Handler) ListAll(w http.ResponseWriter, r *http.Request) {
	docs, err := h.service.ListAll(r.Context())
	if err != nil {
		responses.Error200(w, http.StatusInternalServerError, "Ошибка при получении всех документов: "+err.Error())
		return
	}
	responses.OK(w, map[string]interface{}{"docs": docs})
}

func (h *Handler) Head(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}

	fileID := chi.URLParam(r, "id")

	meta, err := h.service.GetFileMetadata(r.Context(), userID, fileID)
	if err != nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.Header().Set("Content-Disposition", "attachment; filename="+meta.Name)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", meta.Size))
	w.Header().Set("Content-Type", "application/octet-stream")
	w.WriteHeader(http.StatusOK)
}

func (h *Handler) HeadSessionCheck(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		w.WriteHeader(http.StatusOK)
		return
	}
	// Попробуем прогреть кэш списка пользователя
	_, _ = h.service.ListDocumentsForUser(r.Context(), userID, 10, 0)
	w.WriteHeader(http.StatusOK)
}
