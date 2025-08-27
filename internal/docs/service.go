package docs

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"quickdocs/internal/cache"
	"quickdocs/internal/log"
	"time"

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

func (s *Service) CreateDocument(ctx context.Context, doc *Document) error {
	if err := s.repo.Create(ctx, doc); err != nil {
		log.Logger.Printf("req=%s docs.CreateDocument error: id=%s owner=%d err=%v", ctxReqID(ctx), doc.ID, doc.OwnerID, err)
		return err
	}
	_ = s.cache.InvalidateUserFiles(ctx, doc.OwnerID)
	log.Logger.Printf("req=%s docs.CreateDocument ok: id=%s owner=%d", ctxReqID(ctx), doc.ID, doc.OwnerID)
	return nil
}

func (s *Service) GetDocument(ctx context.Context, id uuid.UUID) (*Document, error) {
	return s.repo.Get(ctx, id)
}

func (s *Service) GetDocumentForUser(ctx context.Context, userID int, id uuid.UUID) (*Document, error) {
	if meta, err := s.cache.GetFileMetadata(ctx, userID, id.String()); err == nil && meta != "" {
		var doc Document
		if json.Unmarshal([]byte(meta), &doc) == nil {
			log.Logger.Printf("req=%s docs.GetDocumentForUser cache_hit: user=%d id=%s", ctxReqID(ctx), userID, id)
			return &doc, nil
		}
	}
	doc, err := s.repo.Get(ctx, id)
	if err != nil || doc == nil {
		if err != nil {
			log.Logger.Printf("req=%s docs.GetDocumentForUser repo_err: user=%d id=%s err=%v", ctxReqID(ctx), userID, id, err)
		}
		return doc, err
	}
	if doc.OwnerID != userID && !doc.Public {
		return nil, errors.New("доступ запрещен")
	}
	if b, err := json.Marshal(doc); err == nil {
		_ = s.cache.SetFileMetadata(ctx, userID, id.String(), string(b), time.Minute*10)
	}
	log.Logger.Printf("req=%s docs.GetDocumentForUser repo_ok: user=%d id=%s", ctxReqID(ctx), userID, id)
	return doc, nil
}

func (s *Service) DeleteDocument(ctx context.Context, id uuid.UUID) error {
	doc, _ := s.repo.Get(ctx, id)
	if err := s.repo.Delete(ctx, id); err != nil {
		log.Logger.Printf("req=%s docs.DeleteDocument repo_err: id=%s err=%v", ctxReqID(ctx), id, err)
		return err
	}
	if doc != nil {
		_ = s.cache.InvalidateUserFiles(ctx, doc.OwnerID)
		_ = s.cache.InvalidateFile(ctx, doc.OwnerID, id.String())
	}
	log.Logger.Printf("req=%s docs.DeleteDocument ok: id=%s", ctxReqID(ctx), id)
	return nil
}

func (s *Service) ListAll(ctx context.Context) ([]Document, error) {
	return s.repo.ListAll(ctx)
}
func (s *Service) ListDocuments(ctx context.Context, limit, offset int) ([]*Document, error) {
	return s.repo.List(ctx, limit, offset)
}

func (s *Service) ListDocumentsForUser(ctx context.Context, userID int, limit, offset int) ([]*Document, error) {
	if offset == 0 {
		if cached, err := s.cache.GetUserFiles(ctx, userID); err == nil && cached != "" {
			var items []Document
			if json.Unmarshal([]byte(cached), &items) == nil {
				res := make([]*Document, 0, len(items))
				for i := range items {
					res = append(res, &items[i])
				}
				log.Logger.Printf("req=%s docs.ListDocumentsForUser cache_hit: user=%d", ctxReqID(ctx), userID)
				return res, nil
			}
		}
	}

	docs, err := s.repo.ListForUser(ctx, userID, limit, offset)
	if err != nil {
		log.Logger.Printf("req=%s docs.ListDocumentsForUser repo_err: user=%d err=%v", ctxReqID(ctx), userID, err)
		return nil, err
	}

	if offset == 0 {
		flat := make([]Document, 0, len(docs))
		for _, d := range docs {
			if d != nil {
				flat = append(flat, *d)
			}
		}
		if b, err := json.Marshal(flat); err == nil {
			_ = s.cache.SetUserFiles(ctx, userID, string(b), time.Minute*5)
		}
		log.Logger.Printf("req=%s docs.ListDocumentsForUser repo_ok: user=%d count=%d", ctxReqID(ctx), userID, len(docs))
	}

	return docs, nil
}

func (s *Service) ListDocumentsFiltered(ctx context.Context, userID int, key, value string, limit, offset int, sortBy, order string) ([]*Document, error) {
	return s.repo.ListForUserFiltered(ctx, userID, key, value, limit, offset, sortBy, order)
}

func (s *Service) ListPublicByLogin(ctx context.Context, login string, key, value string, limit, offset int, sortBy, order string) ([]*Document, error) {
	return s.repo.ListPublicByLogin(ctx, login, key, value, limit, offset, sortBy, order)
}

func (s *Service) GetFileMetadata(ctx context.Context, userID int, docID string) (*Document, error) {
	meta, err := s.cache.GetFileMetadata(ctx, userID, docID)
	if err == nil {
		var doc Document
		if err := json.Unmarshal([]byte(meta), &doc); err == nil {
			log.Logger.Printf("req=%s docs.GetFileMetadata cache_hit: user=%d id=%s", ctxReqID(ctx), userID, docID)
			return &doc, nil
		}
	}
	doc, err := s.repo.GetDocumentByID(ctx, docID)
	if err != nil {
		return nil, err
	}

	if doc.OwnerID != userID {
		return nil, errors.New("доступ запрещен")
	}

	data, _ := json.Marshal(doc)
	_ = s.cache.SetFileMetadata(ctx, userID, docID, string(data), time.Minute*10)
	log.Logger.Printf("req=%s docs.GetFileMetadata repo_ok: user=%d id=%s", ctxReqID(ctx), userID, docID)

	return doc, nil
}

func (s *Service) ListFilesByUser(ctx context.Context, userID int) ([]Document, error) {
	data, err := s.cache.GetUserFiles(ctx, userID)
	if err == nil {
		var docs []Document
		if err := json.Unmarshal([]byte(data), &docs); err == nil {
			log.Logger.Printf("req=%s docs.ListFilesByUser cache_hit: user=%d", ctxReqID(ctx), userID)
			return docs, nil
		}
	}

	docs, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	jsonData, _ := json.Marshal(docs)
	_ = s.cache.SetUserFiles(ctx, userID, string(jsonData), time.Minute*5)
	log.Logger.Printf("req=%s docs.ListFilesByUser repo_ok: user=%d count=%d", ctxReqID(ctx), userID, len(docs))

	return docs, nil
}

func (s *Service) GetFileBytesCached(ctx context.Context, userID int, id uuid.UUID, path string, mime string) ([]byte, error) {
	if data, err := s.cache.GetFileBytes(ctx, userID, id.String()); err == nil && len(data) > 0 {
		log.Logger.Printf("req=%s docs.GetFileBytesCached cache_hit: user=%d id=%s", ctxReqID(ctx), userID, id)
		return data, nil
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		log.Logger.Printf("req=%s docs.GetFileBytesCached read_err: user=%d id=%s err=%v", ctxReqID(ctx), userID, id, err)
		return nil, err
	}
	_ = s.cache.SetFileBytes(ctx, userID, id.String(), b, time.Minute*10)
	log.Logger.Printf("req=%s docs.GetFileBytesCached cache_set: user=%d id=%s size=%d", ctxReqID(ctx), userID, id, len(b))
	return b, nil
}
