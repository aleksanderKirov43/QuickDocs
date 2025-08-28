package auth

import (
	"time"

	"quickdocs/internal/cache"
	"quickdocs/internal/users"
)

type Handler struct {
	auth AuthService
}

type Service struct {
	userRepo users.UserRepository
	store    *cache.SessionStore
	tokenTTL time.Duration
}

type User struct {
	ID    int
	Login string
}

type ErrorResponse struct {
	Code int    `json:"code"`
	Text string `json:"text"`
}

type RegisterRequest struct {
	Token string `json:"token"`
	Login string `json:"login"`
	Pswd  string `json:"pswd"`
}

type LoginRequest struct {
	Login string `json:"login"`
	Pswd  string `json:"pswd"`
}

type RegisterResponse struct {
	Response *RegisterData  `json:"response,omitempty"`
	Error    *ErrorResponse `json:"error,omitempty"`
}

type LoginResponse struct {
	Response *LoginData     `json:"response,omitempty"`
	Error    *ErrorResponse `json:"error,omitempty"`
}

type RegisterData struct {
	Login string `json:"login"`
}

type LoginData struct {
	Token string `json:"token"`
}
