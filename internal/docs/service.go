package docs

import (
	"context"
	"encoding/json"
	"errors"
	"quickdocs/internal/cache"
	"time"

	"github.com/google/uuid"
)

type Service struct {
	repo  DocumentRepository
	cache cache.FileCache
}

func NewService(repo DocumentRepository, cache cache.FileCache) *Service {
	return &Service{repo: repo, cache: cache}
}

func (s *Service) CreateDocument(ctx context.Context, doc *Document) error {
	return s.repo.Create(ctx, doc)
}

func (s *Service) GetDocument(ctx context.Context, id uuid.UUID) (*Document, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) DeleteDocument(ctx context.Context, id uuid.UUID) error {
	return s.repo.Delete(ctx, id)
}

func (s *Service) ListAll(ctx context.Context) ([]Document, error) {
	return s.repo.ListAll(ctx)
}
func (s *Service) ListDocuments(ctx context.Context, limit, offset int) ([]*Document, error) {
	return s.repo.List(ctx, limit, offset)
}

func (s *Service) ListDocumentsForUser(ctx context.Context, userID int, limit, offset int) ([]*Document, error) {
	return s.repo.ListForUser(ctx, userID, limit, offset)
}

func (s *Service) GetFileMetadata(ctx context.Context, userID int, docID string) (*Document, error) {
	// Из кеша
	meta, err := s.cache.GetFileMetadata(ctx, userID, docID)
	if err == nil {
		var doc Document
		if err := json.Unmarshal([]byte(meta), &doc); err == nil {
			return &doc, nil
		}
	}
	// Фоллбэк в БД
	doc, err := s.repo.GetDocumentByID(ctx, docID)
	if err != nil {
		return nil, err
	}

	if doc.OwnerID != userID {
		return nil, errors.New("Доступ запрещен")
	}

	data, _ := json.Marshal(doc)
	_ = s.cache.SetFileMetadata(ctx, userID, docID, string(data), time.Minute*10)

	return doc, nil
}

func (s *Service) ListFilesByUser(ctx context.Context, userID int) ([]Document, error) {
	// Попробуй из кеша
	data, err := s.cache.GetUserFiles(ctx, userID)
	if err == nil {
		var docs []Document
		if err := json.Unmarshal([]byte(data), &docs); err == nil {
			return docs, nil
		}
	}

	// Фоллбэк в БД
	docs, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	jsonData, _ := json.Marshal(docs)
	_ = s.cache.SetUserFiles(ctx, userID, string(jsonData), time.Minute*5)

	return docs, nil
}
