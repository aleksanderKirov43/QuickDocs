package app

import (
	"log"
	"net/http"
	"quickdocs/config"
	"quickdocs/internal/router"

	http2 "quickdocs/internal/api/http"
	"quickdocs/internal/repository"
	"quickdocs/internal/services"

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

	sqlPool := repository.NewPostgresPool(cfg.PostgresDSN)
	defer sqlPool.Close()

	userRepo := repository.NewUserRepo(sqlPool)
	docRepo := repository.NewDocsRepository(sqlPool)

	redisStore := repository.NewSessionStore(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB)
	fileCache := repository.NewFileCache(cfg.RedisAddr, cfg.RedisPass, cfg.RedisDB)

	authService := services.NewServiceAuth(userRepo, redisStore)
	docsService := services.NewServiceDocs(docRepo, fileCache)

	authHandler := http2.NewHandlerAuth(authService)
	docsHandler := http2.NewHandlerDocs(docsService, "./uploads")

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
