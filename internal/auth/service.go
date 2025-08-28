package auth

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"quickdocs/internal/cache"
	"quickdocs/internal/users"
	passwords "quickdocs/pkg"

	"github.com/google/uuid"
)

type AuthService interface {
	RegisterUser(ctx context.Context, login, password string) error
	ValidateToken(ctx context.Context, token string) (*User, error)
	Login(ctx context.Context, login, password string) (string, error)
	Logout(ctx context.Context, token string) error
	GenerateToken(login string) (string, error)
	CheckPassword(ctx context.Context, login, password string) (string, error)
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

func NewService(repo users.UserRepository, store *cache.SessionStore) AuthService {
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

func (s *Service) Login(ctx context.Context, login, password string) (string, error) {

	id, hash, err := s.userRepo.GetUserByLogin(ctx, login)
	if err != nil {
		return "", errors.New("Не верный логин или пароль")
	}

	if !passwords.ComparePassword(hash, password) {
		return "", errors.New("Не верный логин или пароль")
	}

	token := uuid.NewString()
	if err := s.store.SaveToken(ctx, token, id, s.tokenTTL); err != nil {
		return "", fmt.Errorf("ошибка сохранения токена в Redis: %w", err)
	}

	return token, nil
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

func (s *Service) CheckPassword(ctx context.Context, login, password string) (string, error) {
	id, hash, err := s.userRepo.GetUserByLogin(ctx, login)
	if err != nil {
		return "", errors.New("Не верный логин или пароль")
	}

	if !passwords.ComparePassword(hash, password){
		return "", errors.New("Не верный логин или пароль")
	}

	token := uuid.NewString()
	if err := s.store.SaveToken(ctx, token, id, s.tokenTTL); err != nil {
		return "", fmt.Errorf("ошибка сохранения токена в Redis: %w", err)
	}

	return token, nil	
}
