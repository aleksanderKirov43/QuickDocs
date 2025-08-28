package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	AppPort     string
	AdminToken  string
	PostgresDSN string
	RedisAddr   string
	RedisPass   string
	RedisDB     int
	AppEnv      string
}

var Cfg Config

func Load() error {

	dbInt, _ := strconv.Atoi(os.Getenv("REDIS_DB"))

	Cfg = Config{
		AppPort:     os.Getenv("APP_PORT"),
		AdminToken:  os.Getenv("ADMIN_TOKEN"),
		PostgresDSN: os.Getenv("POSTGRES_DSN"),
		RedisAddr:   os.Getenv("REDIS_ADDR"),
		RedisPass:   os.Getenv("REDIS_PASSWORD"),
		RedisDB:     dbInt,
		AppEnv:      os.Getenv("APP_ENV"),
	}

	if Cfg.AppPort == "" || Cfg.AdminToken == "" || Cfg.PostgresDSN == "" {
		return fmt.Errorf("Отсутствуют обязательные переменные")
	}

	return nil
}
