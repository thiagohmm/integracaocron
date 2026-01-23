package rabbitmq

import (
	"context"
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/streadway/amqp"
	"github.com/thiagohmm/integracaocron/pkg/logger"
	"github.com/thiagohmm/integracaocron/pkg/tracing"
)

type HTTPServer struct {
	server      *http.Server
	handler     *HTTPHandler
	db          *sql.DB
	rabbitmqURL string
}

type HealthCheckResponse struct {
	Status    string                 `json:"status"`
	Timestamp string                 `json:"timestamp"`
	Checks    map[string]HealthCheck `json:"checks"`
}

type HealthCheck struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

func NewHTTPServer(port string, listener *Listener, db *sql.DB, rabbitmqURL string) *HTTPServer {
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	handler := NewHTTPHandler(listener)

	httpServer := &HTTPServer{
		handler:     handler,
		db:          db,
		rabbitmqURL: rabbitmqURL,
	}

	// Health check endpoint
	router.GET("/health", httpServer.healthCheck)

	// Integration endpoint
	router.POST("/integration", handler.ProcessIntegration)

	server := &http.Server{
		Addr:    ":" + port,
		Handler: router,
	}

	httpServer.server = server

	return httpServer
}

func (s *HTTPServer) healthCheck(c *gin.Context) {
	ctx, span := tracing.StartSpan(c.Request.Context(), "http.health_check")
	defer span.End()

	logger.Debug(ctx, "Health check endpoint accessed")

	response := HealthCheckResponse{
		Status:    "healthy",
		Timestamp: time.Now().Format(time.RFC3339),
		Checks:    make(map[string]HealthCheck),
	}

	// Check database connection
	dbCheck := s.checkDatabase(ctx)
	response.Checks["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		response.Status = "unhealthy"
	}

	// Check RabbitMQ connection
	rabbitmqCheck := s.checkRabbitMQ(ctx)
	response.Checks["rabbitmq"] = rabbitmqCheck
	if rabbitmqCheck.Status != "healthy" {
		response.Status = "unhealthy"
	}

	statusCode := http.StatusOK
	if response.Status == "unhealthy" {
		statusCode = http.StatusServiceUnavailable
	}

	logger.Info(ctx, "Health check completed: %s", response.Status)
	c.JSON(statusCode, response)
}

func (s *HTTPServer) checkDatabase(ctx context.Context) HealthCheck {
	if s.db == nil {
		return HealthCheck{
			Status:  "unhealthy",
			Message: "Database connection not initialized",
		}
	}

	// Ping database with timeout
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := s.db.PingContext(pingCtx)
	if err != nil {
		// Check if it's a broken pipe or connection error
		errStr := err.Error()
		if containsAny(errStr, []string{"broken pipe", "connection reset", "EOF", "bad connection", "invalid connection"}) {
			logger.Warn(ctx, "Database connection broken, forcing pool reset: %v", err)

			// Get current pool stats
			stats := s.db.Stats()
			logger.Info(ctx, "Pool stats before reset - Open: %d, InUse: %d, Idle: %d",
				stats.OpenConnections, stats.InUse, stats.Idle)

			// Strategy: Force close ALL idle connections and create fresh ones
			// Set MaxIdleConns to 0 temporarily to close all idle connections
			s.db.SetMaxIdleConns(0)
			time.Sleep(50 * time.Millisecond)

			// Restore to original value
			s.db.SetMaxIdleConns(25)

			// Also reduce max lifetime temporarily to force rotation
			s.db.SetConnMaxLifetime(1 * time.Nanosecond)
			time.Sleep(50 * time.Millisecond)
			s.db.SetConnMaxLifetime(30 * time.Minute)

			// Now try multiple pings with fresh connections
			var lastErr error
			for i := 0; i < 5; i++ {
				// Each ping should get a fresh connection from the pool
				retryCtx, retryCancel := context.WithTimeout(ctx, 3*time.Second)
				lastErr = s.db.PingContext(retryCtx)
				retryCancel()

				if lastErr == nil {
					newStats := s.db.Stats()
					logger.Info(ctx, "Database connection recovered after %d attempts", i+1)
					logger.Info(ctx, "Pool stats after recovery - Open: %d, InUse: %d, Idle: %d",
						newStats.OpenConnections, newStats.InUse, newStats.Idle)
					return HealthCheck{
						Status:  "healthy",
						Message: "Database connection recovered",
					}
				}

				// Check if error is still connection-related
				if !containsAny(lastErr.Error(), []string{"broken pipe", "connection reset", "EOF"}) {
					// Different error, stop retrying
					break
				}

				// Wait before next retry
				if i < 4 {
					time.Sleep(300 * time.Millisecond)
				}
			}

			logger.Error(ctx, "Database health check failed after 5 retry attempts: %v", lastErr)
			return HealthCheck{
				Status:  "unhealthy",
				Message: "Database connection broken: " + lastErr.Error(),
			}
		}

		logger.Error(ctx, "Database health check failed: %v", err)
		return HealthCheck{
			Status:  "unhealthy",
			Message: "Database ping failed: " + err.Error(),
		}
	}

	return HealthCheck{
		Status:  "healthy",
		Message: "Database connection is active",
	}
}

// containsAny checks if a string contains any of the substrings
func containsAny(s string, substrs []string) bool {
	s = strings.ToLower(s)
	for _, substr := range substrs {
		if strings.Contains(s, strings.ToLower(substr)) {
			return true
		}
	}
	return false
}

func (s *HTTPServer) checkRabbitMQ(ctx context.Context) HealthCheck {
	if s.rabbitmqURL == "" {
		return HealthCheck{
			Status:  "unhealthy",
			Message: "RabbitMQ URL not configured",
		}
	}

	// Try to establish a connection with timeout
	done := make(chan error, 1)
	go func() {
		conn, err := amqp.Dial(s.rabbitmqURL)
		if err != nil {
			done <- err
			return
		}
		defer conn.Close()

		// Try to open a channel to verify connection is fully functional
		ch, err := conn.Channel()
		if err != nil {
			done <- err
			return
		}
		defer ch.Close()

		done <- nil
	}()

	select {
	case err := <-done:
		if err != nil {
			logger.Error(ctx, "RabbitMQ health check failed: %v", err)
			return HealthCheck{
				Status:  "unhealthy",
				Message: "RabbitMQ connection failed: " + err.Error(),
			}
		}
		return HealthCheck{
			Status:  "healthy",
			Message: "RabbitMQ connection is active",
		}
	case <-time.After(5 * time.Second):
		logger.Error(ctx, "RabbitMQ health check timeout")
		return HealthCheck{
			Status:  "unhealthy",
			Message: "RabbitMQ connection timeout",
		}
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
