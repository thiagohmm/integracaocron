package rabbitmq

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/thiagohmm/integracaocron/pkg/logger"
	"github.com/thiagohmm/integracaocron/pkg/tracing"
)

type HTTPServer struct {
	server  *http.Server
	handler *HTTPHandler
}

func NewHTTPServer(port string, listener *Listener) *HTTPServer {
	gin.SetMode(gin.ReleaseMode)
	
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	handler := NewHTTPHandler(listener)

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		ctx, span := tracing.StartSpan(c.Request.Context(), "http.health_check")
		defer span.End()
		logger.Debug(ctx, "Health check endpoint accessed")
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Integration endpoint
	router.POST("/integration", handler.ProcessIntegration)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	return &HTTPServer{
		server:  server,
		handler: handler,
	}
}

func (s *HTTPServer) Start() error {
	ctx := context.Background()
	logger.Info(ctx, "Iniciando servidor HTTP na porta %s", s.server.Addr)
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Stop() error {
	ctx := context.Background()
	logger.Info(ctx, "Parando servidor HTTP...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return s.server.Shutdown(shutdownCtx)
}