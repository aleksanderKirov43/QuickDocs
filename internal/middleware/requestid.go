package middleware

import (
	"crypto/rand"
	"encoding/hex"
)

func generateRequestID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}
