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
	"time"

	"quickdocs/internal/cache"
	"quickdocs/internal/log"

	"github.com/google/uuid"
)

func ctxReqID(ctx context.Context) string {
	if v := ctx.Value("reqID"); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

type Service struct {
	repo  DocumentRepository
	cache cache.FileCache
}

func NewService(repo DocumentRepository, cache cache.FileCache) *Service {
	return &Service{repo: repo, cache: cache}
}

// CreateDocument создаёт новый документ в БД и инвалидирует кэш списка документов пользователя
func (s *Service) CreateDocument(ctx context.Context, doc *Document) error {
	if err := s.repo.Create(ctx, doc); err != nil {
		log.Logger.Printf("req=%s docs.CreateDocument error: id=%s owner=%d err=%v", ctxReqID(ctx), doc.ID, doc.OwnerID, err)
		return err
	}
	_ = s.cache.InvalidateUserFiles(ctx, doc.OwnerID)
	log.Logger.Printf("req=%s docs.CreateDocument ok: id=%s owner=%d", ctxReqID(ctx), doc.ID, doc.OwnerID)
	return nil
}


// Удалить документ пользователя
func (s *Service) DeleteDocumentForUser(ctx context.Context, userID int, id uuid.UUID) error {
	doc, err := s.repo.Get(ctx, id)
	if err != nil || doc == nil {
		return fmt.Errorf("документ не найден")
	}

	if doc.OwnerID != userID {
		return fmt.Errorf("доступ запрещён")
	}

	// Удаляем документ из БД
	if err := s.repo.Delete(ctx, id); err != nil {
		return fmt.Errorf("не удалось удалить документ: %w", err)
	}

	// Удаляем файл с диска, если есть
	if doc.File && doc.FilePath != "" {
		if err := os.Remove(doc.FilePath); err != nil {
			log.Logger.Printf("req=%s docs.DeleteDocumentForUser file_remove_err: user=%d id=%s err=%v", ctxReqID(ctx), userID, id, err)
		}
	}

	// Инвалидируем кэши
	_ = s.cache.InvalidateUserFiles(ctx, doc.OwnerID)
	_ = s.cache.InvalidateFile(ctx, doc.OwnerID, id.String())

	return nil
}

// Возвращаем список документов с поддержкой фильтров и публичных документов
func (s *Service) ListDocuments(ctx context.Context, userID int, filters ListFilters) ([]*Document, error) {
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
			var items []Document
			if json.Unmarshal([]byte(cached), &items) == nil {
				res := make([]*Document, 0, len(items))
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
		flat := make([]Document, 0, len(docs))
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

// GetFileBytes возвращает байты файла с проверкой прав доступа и использованием кэширования
func (s *Service) GetFileBytes(ctx context.Context, userID int, id uuid.UUID) ([]byte, error) {
	// Получаем документ с проверкой прав
	doc, err := s.repo.Get(ctx, id)
	if err != nil || doc == nil {
		return nil, fmt.Errorf("документ не найден")
	}

	// Проверяем права доступа
	if doc.OwnerID != userID && !doc.Public {
		return nil, errors.New("доступ запрещен")
	}

	if !doc.File || doc.FilePath == "" {
		return nil, fmt.Errorf("файл не найден")
	}

	// Пробуем получить из кэша
	if data, err := s.cache.GetFileBytes(ctx, userID, id.String()); err == nil && len(data) > 0 {
		log.Logger.Printf("req=%s docs.GetFileBytes cache_hit: user=%d id=%s", ctxReqID(ctx), userID, id)
		return data, nil
	}

	// Читаем с диска
	b, err := os.ReadFile(doc.FilePath)
	if err != nil {
		log.Logger.Printf("req=%s docs.GetFileBytes read_err: user=%d id=%s err=%v", ctxReqID(ctx), userID, id, err)
		return nil, err
	}

	// Сохраняем в кэш
	_ = s.cache.SetFileBytes(ctx, userID, id.String(), b, time.Minute*10)
	log.Logger.Printf("req=%s docs.GetFileBytes cache_set: user=%d id=%s size=%d", ctxReqID(ctx), userID, id, len(b))

	return b, nil
}
