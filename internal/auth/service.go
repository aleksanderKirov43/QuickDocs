package auth

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"quickdocs/config"
	"quickdocs/internal/cache"
	"quickdocs/internal/db"
	passwords "quickdocs/pkg"

	"github.com/google/uuid"
)

type Service struct {
	userRepo db.UserRepository
	store    *cache.SessionStore
	tokenTTL time.Duration
}

type User struct {
	ID    int
	Login string
}

type AuthService interface {
	RegisterUser(ctx context.Context, login, password string) error
	ValidateToken(ctx context.Context, token string) (*User, error)
	Logout(ctx context.Context, token string) error
	GenerateToken(login string) (string, error)
}

func NewService(repo db.UserRepository, store *cache.SessionStore) *Service {
	return &Service{
		userRepo: repo,
		store:    store,
		tokenTTL: 24 * time.Hour,
	}
}

// Создаём нового пользователя
func (s *Service) RegisterUser(ctx context.Context, login, password string) error {
	// login: мин. 8, латиница и цифры
	if len(login) < 8 {
		return fmt.Errorf("логин должен быть не короче 8 символов")
	}
	if !regexp.MustCompile(`^[A-Za-z0-9]+$`).MatchString(login) {
		return fmt.Errorf("логин должен содержать только латиницу и цифры")
	}

	if err := passwords.ValidatePassword(password); err != nil {
		return err
	}

	hash, err := passwords.HashPassword(password)
	if err != nil {
		return fmt.Errorf("ошибка хэша пароля: %w", err)
	}

	err = s.userRepo.CreateUser(ctx, login, hash)
	if err != nil {
		return fmt.Errorf("пользователь уже существует или ошибка БД: %w", err)
	}

	return nil
}

// Проверяем токен и возвращаем login пользователя
func (s *Service) ValidateToken(ctx context.Context, token string) (*User, error) {
	userID, err := s.store.GetUserID(ctx, token)
	if err != nil {
		return nil, errors.New("недействительный или просроченный токен")
	}

	login, err := s.userRepo.GetLoginByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return &User{
		ID:    userID,
		Login: login,
	}, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	return s.store.DeleteToken(ctx, token)
}

func (s *Service) GenerateToken(login string) (string, error) {
	token := uuid.NewString()

	userID, _, err := s.userRepo.GetUserByLogin(context.Background(), login)
	if err != nil {
		return "", fmt.Errorf("пользователь не найден: %w", err)
	}

	err = s.store.SaveToken(context.Background(), token, userID, s.tokenTTL)
	if err != nil {
		return "", fmt.Errorf("ошибка сохранения токена в Redis: %w", err)
	}

	return token, nil
}

// CheckAdminToken сверяет переданный токен с конфигом
func (s *Service) CheckAdminToken(ctx context.Context, token string) error {
	if token == "" || token != config.Cfg.AdminToken {
		return errors.New("недействительный токен администратора")
	}
	return nil
}
