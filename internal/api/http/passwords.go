package http

import (
	"errors"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

func ValidatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("Пароль должен содержать не менее 8 символов")
	}

	var (
		lowerCount   int
		upperCount   int
		digitCount   int
		specialCount int
	)

	for _, r := range password {
		switch {
		case unicode.IsLower(r):
			lowerCount++
		case unicode.IsUpper(r):
			upperCount++
		case unicode.IsDigit(r):
			digitCount++
		case unicode.IsPunct(r) || unicode.IsSymbol(r):
			specialCount++
		}
	}

	if lowerCount == 0 {
		return errors.New("Пароль должен содержать минимум одну строчную букву")
	}

	if upperCount == 0 {
		return errors.New("Пароль должен содержать минимум одну заглавную букву")
	}

	if digitCount == 0 {
		return errors.New("Пароль должен содержать минимум одну цифру")
	}

	if specialCount == 0 {
		return errors.New("Пароль должен содержать минимум один специальный символ")
	}

	if lowerCount+upperCount < 2 {
		return errors.New("Пароль должен содержать минимум две буквы в разных регистрах")
	}

	return nil
}

func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(hash), err
}

func ComparePassword(hash, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
