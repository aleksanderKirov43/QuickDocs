package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"quickdocs/internal/models"
	"quickdocs/internal/repository"
	"time"

	"quickdocs/internal/log"

	"github.com/google/uuid"
)

type ctxKey string

const reqIDKey ctxKey = "reqID"

// Для добавления уникального ID на каждый HTTP в контекст
func ctxReqID(ctx context.Context) string {
	if v := ctx.Value(reqIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

type ServiceDocs struct {
	repo  repository.DocumentRepository
	cache repository.FileCache
}

func NewServiceDocs(repo repository.DocumentRepository, cache repository.FileCache) *ServiceDocs {
	return &ServiceDocs{repo: repo, cache: cache}
}

func (s *ServiceDocs) CreateDocument(ctx context.Context, userID int, file io.Reader, fileName, meta string) (*models.Document, error) {
	// Разбираем meta JSON
	var jsonData *json.RawMessage
	public := false
	if meta != "" {
		var metaMap map[string]interface{}
		if err := json.Unmarshal([]byte(meta), &metaMap); err != nil {
			return nil, fmt.Errorf("неверный формат meta JSON: %w", err)
		}
		jm := json.RawMessage(meta)
		jsonData = &jm

		if pub, ok := metaMap["public"].(bool); ok {
			public = pub
		}
	}

	fileID := uuid.New()
	uploadDir := "./uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		return nil, fmt.Errorf("невозможно создать директорию: %w", err)
	}
	filePath := filepath.Join(uploadDir, fileID.String()+"_"+fileName)

	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("невозможно сохранить файл: %w", err)
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		return nil, fmt.Errorf("ошибка записи файла: %w", err)
	}

	doc := &models.Document{
		ID:       fileID,
		OwnerID:  userID,
		Name:     fileName,
		File:     true,
		Public:   public,
		JsonData: jsonData,
	}

	if err := s.repo.Create(ctx, doc); err != nil {
		return nil, fmt.Errorf("не удалось создать запись документа: %w", err)
	}

	_ = s.cache.InvalidateUserFiles(ctx, userID)
	log.Logger.Printf("req=%s docs.CreateDocument ok: id=%s owner=%d", ctxReqID(ctx), doc.ID, userID)

	return doc, nil
}

// Удалить документ пользователя
func (s *ServiceDocs) DeleteDocumentForUser(ctx context.Context, userID int, id uuid.UUID) error {
	doc, err := s.repo.Get(ctx, id)
	if err != nil || doc == nil {
		return fmt.Errorf("документ не найден")
	}

	if doc.OwnerID != userID {
		return fmt.Errorf("доступ запрещён")
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("не удалось удалить документ: %w", err)
	}

	// Удаляем файл с диска, если есть
	if doc.File {
		filePath := fmt.Sprintf("./uploads/%s_%s", doc.ID.String(), doc.Name)
		if err := os.Remove(filePath); err != nil {
			log.Logger.Printf("req=%s docs.DeleteDocumentForUser file_remove_err: user=%d id=%s err=%v",
				ctxReqID(ctx), userID, doc.ID, err)
		}
	}

	_ = s.cache.InvalidateUserFiles(ctx, doc.OwnerID)
	_ = s.cache.InvalidateFile(ctx, doc.OwnerID, id.String())

	return nil
}

// Возвращаем список документов с поддержкой фильтров и публичных документов
func (s *ServiceDocs) ListDocuments(ctx context.Context, userID int, filters models.ListFilters) ([]*models.Document, error) {
	if filters.Login != "" {
		// Публичные документы другого пользователя
		return s.repo.ListPublicByLogin(ctx, filters.Login, filters.Key, filters.Value, filters.Limit, filters.Offset, filters.SortBy, filters.Order)
	}

	if filters.Key != "" || filters.Value != "" || filters.SortBy != "" || filters.Order != "" {
		// Фильтрованный список
		return s.repo.ListForUserFiltered(ctx, userID, filters.Key, filters.Value, filters.Limit, filters.Offset, filters.SortBy, filters.Order)
	}

	// Обычный список пользователя с кэшированием первой страницы
	if filters.Offset == 0 {
		if cached, err := s.cache.GetUserFiles(ctx, userID); err == nil && cached != "" {
			var items []models.Document
			if json.Unmarshal([]byte(cached), &items) == nil {
				res := make([]*models.Document, 0, len(items))
				for i := range items {
					res = append(res, &items[i])
				}
				log.Logger.Printf("req=%s docs.ListDocuments cache_hit: user=%d", ctxReqID(ctx), userID)
				return res, nil
			}
		}
	}

	// Получаем из БД
	docs, err := s.repo.ListForUser(ctx, userID, filters.Limit, filters.Offset)
	if err != nil {
		log.Logger.Printf("req=%s docs.ListDocuments repo_err: user=%d err=%v", ctxReqID(ctx), userID, err)
		return nil, err
	}

	// Кэшируем первую страницу
	if filters.Offset == 0 {
		flat := make([]models.Document, 0, len(docs))
		for _, d := range docs {
			if d != nil {
				flat = append(flat, *d)
			}
		}
		if b, err := json.Marshal(flat); err == nil {
			_ = s.cache.SetUserFiles(ctx, userID, string(b), time.Minute*5)
		}
		log.Logger.Printf("req=%s docs.ListDocuments repo_ok: user=%d count=%d", ctxReqID(ctx), userID, len(docs))
	}

	return docs, nil
}

// Возвращаем файл с проверкой прав доступа и использованием кэширования
func (s *ServiceDocs) GetDocument(ctx context.Context, userID int, id uuid.UUID) (*models.DocumentWithBytes, error) {
	// Получаем документ с проверкой прав
	doc, err := s.repo.Get(ctx, id)
	if err != nil || doc == nil {
		return nil, fmt.Errorf("документ не найден")
	}

	if doc.OwnerID != userID && !doc.Public {
		return nil, errors.New("доступ запрещен")
	}

	if !doc.File {
		return nil, fmt.Errorf("файл не найден")
	}

	// Пробуем получить из кэша
	if data, err := s.cache.GetFileBytes(ctx, userID, id.String()); err == nil && len(data) > 0 {
		log.Logger.Printf("req=%s docs.GetFileBytes cache_hit: user=%d id=%s", ctxReqID(ctx), userID, id)
		return &models.DocumentWithBytes{Data: data, Name: doc.Name}, nil
	}

	// Читаем с диска
	filePath := fmt.Sprintf("./uploads/%s_%s", doc.ID.String(), doc.Name)
	b, err := os.ReadFile(filePath)
	if err != nil {
		log.Logger.Printf("req=%s docs.GetFileBytes read_err: user=%d id=%s err=%v", ctxReqID(ctx), userID, id, err)
		return nil, err
	}

	// Сохраняем в кэш
	_ = s.cache.SetFileBytes(ctx, userID, id.String(), b, time.Minute*10)
	log.Logger.Printf("req=%s docs.GetFileBytes cache_set: user=%d id=%s size=%d", ctxReqID(ctx), userID, id, len(b))

	return &models.DocumentWithBytes{Data: b, Name: doc.Name}, nil
}
