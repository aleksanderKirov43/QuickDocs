package config

import (
	"fmt"
	"os"
)

type Config struct {
	AppPort     string
	AdminToken  string
	PostgresDSN string
	RedisAddr   string
	RedisPass   string
	RedisDB     string
	AppEnv      string
}

var Cfg Config

func Load() error {
	Cfg = Config{
		AppPort:     os.Getenv("APP_PORT"),
		AdminToken:  os.Getenv("ADMIN_TOKEN"),
		PostgresDSN: os.Getenv("POSTGRES_DSN"),
		RedisAddr:   os.Getenv("REDIS_ADDR"),
		RedisPass:   os.Getenv("REDIS_PASSWORD"),
		RedisDB:     os.Getenv("REDIS_DB"),
		AppEnv:      os.Getenv("APP_ENV"),
	}

	if Cfg.AppPort == "" || Cfg.AdminToken == "" || Cfg.PostgresDSN == "" {
		return fmt.Errorf("Отсутствуют обязательные переменные")
	}

	return nil
}
