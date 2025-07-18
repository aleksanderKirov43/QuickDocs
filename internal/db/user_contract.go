package db

import (
	"context"
)

type UserRepository interface {
	CreateUser(ctx context.Context, login string, password string) error
	GetUserByLogin(ctx context.Context, login string) (int, string, error)
	GetLoginByID(ctx context.Context, id int) (string, error)
}
