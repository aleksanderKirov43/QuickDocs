package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"quickdocs/internal/cache"
)

type fakeUserRepo struct {
	loginToID map[string]int
	idToLogin map[int]string
}

func (f *fakeUserRepo) CreateUser(ctx context.Context, login string, password string) error {
	return nil
}
func (f *fakeUserRepo) GetUserByLogin(ctx context.Context, login string) (int, string, error) {
	if id, ok := f.loginToID[login]; ok {
		return id, "$2a$10$abcdefghijklmnopqrstuv", nil
	}
	return 0, "", errors.New("not found")
}
func (f *fakeUserRepo) GetLoginByID(ctx context.Context, id int) (string, error) {
	return f.idToLogin[id], nil
}

type fakeStore struct{ cache.FileCache }

func TestGenerateValidateLogout(t *testing.T) {
	store := cache.NewSessionStore("localhost:6379", "", 0)
	repo := &fakeUserRepo{loginToID: map[string]int{"user": 1}, idToLogin: map[int]string{1: "user"}}
	s := NewService(repo, store)
	s.(*Service).tokenTTL = time.Second

	tok, err := s.GenerateToken("user")
	if err != nil || tok == "" {
		t.Fatalf("token gen failed: %v", err)
	}

	usr, err := s.ValidateToken(context.Background(), tok)
	if err != nil || usr == nil || usr.Login != "user" {
		t.Fatalf("validate failed: %v", err)
	}

	if err := s.Logout(context.Background(), tok); err != nil {
		t.Fatalf("logout failed: %v", err)
	}
}
