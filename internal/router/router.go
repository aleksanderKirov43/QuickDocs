package router

import (
	"net/http"
	"quickdocs/app"
	http2 "quickdocs/internal/api/http"
	"quickdocs/pkg"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

func New(authH *http2.AuthHandler, docsH *http2.DocsHandler) http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// публичные
	r.With(pkg.AdminToken).Post("/api/register", authH.Register)
	r.Post("/api/auth", authH.Login)

	// защищённые
	r.Group(func(pr chi.Router) {
		pr.Use(app.NewAuthMiddleware(authH.Auth).Auth)

		pr.Delete("/api/auth/{token}", authH.Logout)

		pr.Route("/api/docs", func(dr chi.Router) {
			dr.Get("/", docsH.List) // список (свой/по фильтрам)
			dr.Head("/", docsH.HeadSessionCheck)
			dr.Post("/", docsH.Upload)               // загрузка
			dr.Get("/{id}", docsH.Get)               // получение документа
			dr.Head("/{id}", docsH.HeadSessionCheck) // HEAD документа
			dr.Delete("/{id}", docsH.Delete)
		})
	})

	return r
}
