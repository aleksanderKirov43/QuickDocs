package users

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type UserRepository interface {
	CreateUser(ctx context.Context, login string, password string) error
	GetUserByLogin(ctx context.Context, login string) (int, string, error)
	GetLoginByID(ctx context.Context, id int) (string, error)
}

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) UserRepository {
	return &UserRepo{db: db}
}

func (r *UserRepo) CreateUser(ctx context.Context, login, password string) error {
	query := `
		INSERT INTO users (login, password)
		VALUES ($1, $2)
	`

	_, err := r.db.ExecContext(ctx, query, login, password)

	if err != nil {
		if isUniqueViolation(err) {
			return fmt.Errorf("Пользователь уже существует")
		}
		return err
	}

	return nil
}

// Возвращаем ID и hash пароля по логину
func (r *UserRepo) GetUserByLogin(ctx context.Context, login string) (int, string, error) {
	query := `
		SELECT id, password FROM users
		WHERE login = $1
	`

	var id int
	var hash string

	err := r.db.QueryRowContext(ctx, query, login).Scan(&id, &hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", fmt.Errorf("Пользователь не найден")
		}
		return 0, "", err
	}

	return id, hash, nil
}

func (r *UserRepo) GetLoginByID(ctx context.Context, id int) (string, error) {
	var login string
	query := `SELECT login FROM users WHERE id = $1`
	err := r.db.QueryRowContext(ctx, query, id).Scan(&login)
	if err != nil {
		return "", err
	}
	return login, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "Повторяющееся значение ключа")
}
