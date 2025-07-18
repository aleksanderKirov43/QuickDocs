package router

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"net/http"

	"quickdocs/internal/auth"
	"quickdocs/internal/docs"
	appmw "quickdocs/internal/middleware"
)

func New(authH *auth.Handler, docsH *docs.Handler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// публичные
	r.Post("/api/register", authH.Register)
	r.Post("/api/auth", authH.Login)

	// защищённые
	r.Group(func(pr chi.Router) {
		pr.Use(appmw.NewAuthMiddleware(authH.Service).Auth)

		pr.Delete("/api/auth/{token}", authH.Logout)

		pr.Route("/api/docs", func(dr chi.Router) {
			dr.Get("/", docsH.ListAll)               // Получение всех документов
			dr.Get("/user/{userID}", docsH.List)     // Получение документов отдельного пользователя
			dr.Head("/user", docsH.HeadSessionCheck) // HEAD для проверки авторизации
			dr.Get("/{id}", docsH.Get)               // Получение документа по id
			dr.Head("/{id}", docsH.Head)             // HEAD для проверки существования документа по id
			dr.Post("/upload", docsH.Upload)         // Загрузка документа
			dr.Delete("/{id}", docsH.Delete)         // Удаление документа
		})
	})

	return r
}
