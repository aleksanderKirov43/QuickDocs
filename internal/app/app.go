package app

import (
	"database/sql"
	"log"
	"net/http"
	"quickdocs/config"
	"quickdocs/internal/auth"
	"quickdocs/internal/cache"
	"quickdocs/internal/docs"
	"quickdocs/internal/router"
	"quickdocs/internal/users"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func Run() {
	if err := godotenv.Load("./config/.env"); err != nil {
		log.Println("Файл .env не найден")
	}

	if err := config.Load(); err != nil {
		log.Fatalf("Ошибка загрузки конфигурации: %v", err)
	}
	cfg := config.Cfg

	// Открытие подключения к PostgreSQL
	sqlDB, err := sql.Open("pgx", cfg.PostgresDSN)
	if err != nil {
		log.Fatalf("Ошибка подключения к базе данных: %v", err)
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		log.Fatalf("База данных недоступна: %v", err)
	}

	// Сделать подключение через pgx. Отдельно в каждом repo

	userRepo := users.NewUserRepo(sqlDB)
	docRepo := docs.NewDocsRepository(sqlDB)

	redisStore := cache.NewSessionStore(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB)
	fileCache := cache.NewFileCache(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB)

	authService := auth.NewService(userRepo, redisStore)
	docsService := docs.NewService(docRepo, fileCache)

	authHandler := auth.NewHandler(authService)
	docsHandler := docs.NewHandler(docsService, "./uploads")

	r := router.New(authHandler, docsHandler)

	port := cfg.AppPort
	if port == "" {
		port = "8080"
	}

	log.Printf("Сервер запущен на порту %s...", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Fatalf("Ошибка запуска сервера: %v", err)
	}
}
