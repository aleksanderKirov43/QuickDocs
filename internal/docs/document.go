package docs

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

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
