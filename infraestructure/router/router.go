package router

import (
	"github.com/thiagohmm/integracaocron/internal/domain/handler" // Ajuste o path

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// SetupRoutes configura todas as rotas da aplicação.
func SetupRoutes(
	r *chi.Mux,

	healthHandler *handler.HealthHandler,

) {
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	// Health Check
	r.Get("/health", healthHandler.Check) // O HealthCheckHandler original era uma função.
}
