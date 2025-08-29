package tests

import (
	"context"
	"encoding/json"
	"quickdocs/internal/models"
	"quickdocs/internal/services"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCacheListForUser(t *testing.T) {
	ctx := context.Background()
	fc := newTestFileCache()
	repo := &stubRepo{list: []*models.Document{{OwnerID: 1, Name: "a"}}}
	svc := services.NewService(repo, fc)

	// первая выборка — попадает в БД и кэширует
	if _, err := svc.ListDocuments(ctx, 1, models.ListFilters{}); err != nil {
		t.Fatal(err)
	}
	// вторая — из кэша
	if _, err := svc.ListDocuments(ctx, 1, models.ListFilters{}); err != nil {
		t.Fatal(err)
	}
}

type stubRepo struct{ list []*models.Document }

func (s *stubRepo) Create(ctx context.Context, doc *models.Document) error          { return nil }
func (s *stubRepo) Get(ctx context.Context, id uuid.UUID) (*models.Document, error) { return nil, nil }
func (s *stubRepo) Delete(ctx context.Context, id uuid.UUID) error                  { return nil }
func (s *stubRepo) List(ctx context.Context, limit, offset int) ([]*models.Document, error) {
	return s.list, nil
}
func (s *stubRepo) ListForUser(ctx context.Context, userID, limit, offset int) ([]*models.Document, error) {
	return s.list, nil
}
func (s *stubRepo) ListAll(ctx context.Context) ([]models.Document, error) { return nil, nil }
func (s *stubRepo) ListByUser(ctx context.Context, userID int) ([]models.Document, error) {
	return nil, nil
}
func (s *stubRepo) GetDocumentByID(ctx context.Context, docID string) (*models.Document, error) {
	return nil, nil
}
func (s *stubRepo) ListForUserFiltered(ctx context.Context, userID int, key, value string, limit, offset int, sortBy, order string) ([]*models.Document, error) {
	return s.list, nil
}
func (s *stubRepo) ListPublicByLogin(ctx context.Context, login string, key, value string, limit, offset int, sortBy, order string) ([]*models.Document, error) {
	return s.list, nil
}

// простейший in-memory FileCache для теста
type memCache struct {
	files map[string]string
	lists map[int]string
}

func newTestFileCache() *memCache {
	return &memCache{files: map[string]string{}, lists: map[int]string{}}
}
func (m *memCache) SetFileMetadata(ctx context.Context, userID int, docID string, meta string, ttl time.Duration) error {
	m.files[docID] = meta
	return nil
}
func (m *memCache) GetFileMetadata(ctx context.Context, userID int, docID string) (string, error) {
	return m.files[docID], nil
}
func (m *memCache) InvalidateFile(ctx context.Context, userID int, docID string) error {
	delete(m.files, docID)
	return nil
}
func (m *memCache) SetUserFiles(ctx context.Context, userID int, data string, ttl time.Duration) error {
	m.lists[userID] = data
	return nil
}
func (m *memCache) GetUserFiles(ctx context.Context, userID int) (string, error) {
	return m.lists[userID], nil
}
func (m *memCache) InvalidateUserFiles(ctx context.Context, userID int) error {
	delete(m.lists, userID)
	return nil
}
func (m *memCache) SetFileBytes(ctx context.Context, userID int, docID string, data []byte, ttl time.Duration) error {
	b, _ := json.Marshal(data)
	m.files[docID] = string(b)
	return nil
}
func (m *memCache) GetFileBytes(ctx context.Context, userID int, docID string) ([]byte, error) {
	var out []byte
	_ = json.Unmarshal([]byte(m.files[docID]), &out)
	return out, nil
}
