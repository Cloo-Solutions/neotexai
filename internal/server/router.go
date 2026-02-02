package server

import (
	"net/http"

	"github.com/cloo-solutions/neotexai/internal/api"
	"github.com/cloo-solutions/neotexai/internal/api/handlers"
	"github.com/cloo-solutions/neotexai/internal/api/middleware"
	"github.com/go-chi/chi/v5"
)

type RouterConfig struct {
	AuthValidator    middleware.AuthValidator
	KnowledgeHandler *handlers.KnowledgeHandler
	AssetHandler     *handlers.AssetHandler
	ContextHandler   *handlers.ContextHandler
	AuthHandler      *handlers.AuthHandler
	ProjectHandler   *handlers.ProjectHandler
}

func NewRouter(cfg RouterConfig) http.Handler {
	r := chi.NewRouter()

	const maxBodyBytes int64 = 5 * 1024 * 1024

	r.Use(middleware.RequestID)
	r.Use(middleware.SentryMiddleware)
	r.Use(middleware.AccessLog)
	r.Use(middleware.MaxBodyBytes(maxBodyBytes))

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		api.Success(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	r.Group(func(r chi.Router) {
		r.Use(middleware.APIKeyAuth(cfg.AuthValidator))

		r.Route("/knowledge", func(r chi.Router) {
			r.Post("/", cfg.KnowledgeHandler.Create)
			r.Get("/", cfg.KnowledgeHandler.List)
			r.Get("/{id}", cfg.KnowledgeHandler.Get)
			r.Put("/{id}", cfg.KnowledgeHandler.Update)
			r.Delete("/{id}", cfg.KnowledgeHandler.Delete)
		})

		r.Route("/assets", func(r chi.Router) {
			r.Post("/init", cfg.AssetHandler.InitUpload)
			r.Post("/complete", cfg.AssetHandler.CompleteUpload)
			r.Get("/{id}/download", cfg.AssetHandler.GetDownloadURL)
		})

		r.Get("/context", cfg.ContextHandler.GetManifest)
		r.Post("/search", cfg.ContextHandler.Search)
		r.Post("/search/feedback", cfg.ContextHandler.SearchFeedback)

		r.Route("/projects", func(r chi.Router) {
			r.Post("/", cfg.ProjectHandler.Create)
			r.Get("/", cfg.ProjectHandler.List)
			r.Get("/{id}", cfg.ProjectHandler.Get)
		})
	})

	r.Post("/orgs", cfg.AuthHandler.CreateOrg)
	r.Post("/apikeys", cfg.AuthHandler.CreateAPIKey)

	return r
}
