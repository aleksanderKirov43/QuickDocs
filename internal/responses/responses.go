package responses

import (
	"encoding/json"
	"net/http"
)

type Error struct {
	Code int    `json:"code"`
	Text string `json:"text"`
}

type Envelope struct {
	Error    *Error      `json:"error,omitempty"`
	Response interface{} `json:"response,omitempty"`
	Data     interface{} `json:"data,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, env Envelope) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(env)
}

func OK(w http.ResponseWriter, payload interface{}) {
	WriteJSON(w, http.StatusOK, Envelope{Data: payload})
}

func Ack(w http.ResponseWriter, payload interface{}) {
	WriteJSON(w, http.StatusOK, Envelope{Response: payload})
}

func Created(w http.ResponseWriter, payload interface{}) {
	WriteJSON(w, http.StatusOK, Envelope{Data: payload})
}

func Fail(w http.ResponseWriter, status int, code int, msg string) {
	if w != nil {
		w.Header().Set("Content-Type", "application/json")
	}
	if status == 0 {
		status = http.StatusOK
	}
	if rw, ok := w.(interface {
		Header() http.Header
		WriteHeader(statusCode int)
	}); ok {
		rw.WriteHeader(status)
	}
	if status == http.StatusOK {
		_ = json.NewEncoder(w).Encode(Envelope{Error: &Error{Code: code, Text: msg}})
	}
}

// Error200 записывает ошибку в поле error и HTTP 200
func Error200(w http.ResponseWriter, code int, msg string) {
	WriteJSON(w, http.StatusOK, Envelope{Error: &Error{Code: code, Text: msg}})
}
