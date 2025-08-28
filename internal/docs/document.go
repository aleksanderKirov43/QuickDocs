package docs

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Основная структура документа
type Document struct {
	ID       uuid.UUID        `json:"id"`
	OwnerID  int              `json:"owner_id"`
	Name     string           `json:"name"`
	Mime     string           `json:"mime,omitempty"`
	File     bool             `json:"has_file"`
	Public   bool             `json:"public"`
	Created  time.Time        `json:"created_at"`
	JsonData *json.RawMessage `json:"json,omitempty"`
	FilePath string           `json:"-"`
	Size     int64            `json:"size"`
}

// Фильтры для списка документов
type ListFilters struct {
	Limit, Offset int
	Key, Value    string
	SortBy, Order string
	Login         string // для публичных документов
}

// Данные для загрузки документа
type UploadRequest struct {
	Meta     *Document        `json:"meta"`
	JsonData *json.RawMessage `json:"json,omitempty"`
	File     []byte           `json:"file,omitempty"`
	FileName string           `json:"file_name,omitempty"`
}
