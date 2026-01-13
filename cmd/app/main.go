package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"

	"github.com/thiagohmm/integracaocron/configuration"
	"github.com/thiagohmm/integracaocron/domain/repositories"
	"github.com/thiagohmm/integracaocron/domain/usecases"
	"github.com/thiagohmm/integracaocron/infraestructure/database"
	rabbitmq "github.com/thiagohmm/integracaocron/internal/delivery"
	"github.com/thiagohmm/integracaocron/pkg/logger"
	"github.com/thiagohmm/integracaocron/pkg/tracing"
)

func main() {
	ctx := context.Background()
	log.Println("=== Iniciando aplicação IntegracaoCron ===")

	// Load configuration
	cfg, err := loadConfiguration()
	if err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
	}

	// Initialize tracing
	err = initTracing(cfg)
	if err != nil {
		log.Printf("Erro ao inicializar tracing: %v", err)
	}
	defer func() {
		if err := tracing.Shutdown(ctx); err != nil {
			log.Printf("Erro ao finalizar tracing: %v", err)
		}
	}()

	logger.Info(ctx, "Aplicação IntegracaoCron iniciada com tracing")

	// Connect to database
	db, err := database.ConectarBanco(cfg)
	if err != nil {
		logger.Error(ctx, "Erro ao conectar ao banco de dados: %v", err)
		log.Fatalf("Erro ao conectar ao banco de dados: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			logger.Error(ctx, "Erro ao fechar conexão com o banco: %v", err)
		}
	}()

	logger.Info(ctx, "Conexão com banco de dados estabelecida")

	// Initialize repositories
	promotionRepo := repositories.NewPromotionRepository(db)
	parameterRepo := repositories.NewParameterRepository(db)
	integrationRepo := repositories.NewIntegrationRepository(db)
	networkRepo := repositories.NewNetworkRepository(db)
	productIntegrationRepo := repositories.NewProductIntegrationRepository(db)
	promotionNormalizationRepo := repositories.NewPromotionNormalizationRepository(db)
	productRepo := repositories.NewProductRepository(db)
	productPackagingRepo := repositories.NewProductPackagingRepository(db)
	unitOfMeasurementRepo := repositories.NewUnitOfMeasurementRepository(db)
	packagingIntegrationStagingRepo := repositories.NewPackagingIntegrationStagingRepository(db)


	// Get RabbitMQ configuration
	rabbitmqURL := getRabbitMQURL(cfg)
	if rabbitmqURL == "" {
		logger.Error(ctx, "URL do RabbitMQ não configurada")
		log.Fatal("URL do RabbitMQ não configurada")
	}

	logger.Info(ctx, "Configuração RabbitMQ carregada: %s", maskRabbitMQURL(rabbitmqURL))

	// Initialize use cases
	packagingIntegrationUC := usecases.NewPackagingIntegrationUseCase(productRepo, productPackagingRepo, unitOfMeasurementRepo, packagingIntegrationStagingRepo)
	productIntegrationUC := usecases.NewProductIntegrationUseCase(productIntegrationRepo, packagingIntegrationUC, db)
	integrationJobUC := usecases.NewIntegrationJobUseCase(parameterRepo, integrationRepo, networkRepo, productIntegrationUC, db)
	promotionUC := usecases.NewPromotionUseCase(promotionRepo, rabbitmqURL, integrationJobUC)
	promotionNormalizationUC := usecases.NewPromotionNormalizationUseCase(promotionNormalizationRepo, db)

	// Get number of workers from environment or use default
	workers := getWorkersCount()
	logger.Info(ctx, "Número de workers configurado: %d", workers)

	// Initialize RabbitMQ listener
	listener := &rabbitmq.Listener{
		PromocaoUC:               promotionUC,
		IntegrationUc:            integrationJobUC,
		ProductIntegrationUC:     productIntegrationUC,
		PromotionNormalizationUC: promotionNormalizationUC,
		Workers:                  workers,
	}

	logger.Info(ctx, "Use cases inicializados com sucesso")

	// Setup graceful shutdown
	setupGracefulShutdown()

	// Get HTTP port
	httpPort := getHTTPPort(cfg)

	// Start HTTP server in a goroutine
	var wg sync.WaitGroup
	if httpPort != "" {
		wg.Add(1)
		go func() {
			defer wg.Done()
			httpServer := rabbitmq.NewHTTPServer(httpPort, listener)
			logger.Info(ctx, "Iniciando servidor HTTP na porta %s", httpPort)
			if err := httpServer.Start(); err != nil {
				logger.Error(ctx, "Erro no servidor HTTP: %v", err)
			}
		}()
	}

	// Start listening to RabbitMQ
	logger.Info(ctx, "Iniciando listener RabbitMQ com %d workers", workers)
	logger.Info(ctx, "Conectando ao RabbitMQ: %s", maskRabbitMQURL(rabbitmqURL))

	if err := listener.ListenToQueue(rabbitmqURL); err != nil {
		logger.Error(ctx, "Erro ao iniciar listener RabbitMQ: %v", err)
		log.Fatalf("Erro ao iniciar listener RabbitMQ: %v", err)
	}
}

// initTracing initializes distributed tracing
func initTracing(cfg *configuration.Conf) error {
	config := tracing.Config{
		ServiceName:    "integracaocron",
		ServiceVersion: "1.0.0",
		Environment:    "production",
		JaegerEndpoint: cfg.JaegerEndpoint,
		Enabled:        cfg.TracingEnabled,
		SamplingRate:   cfg.TracingSampleRate,
	}

	if config.JaegerEndpoint == "" {
		config.JaegerEndpoint = "http://localhost:14268/api/traces"
	}
	if config.SamplingRate == 0 {
		config.SamplingRate = 1.0
	}

	return tracing.InitTracer(config)
}

// loadConfiguration loads the application configuration
func loadConfiguration() (*configuration.Conf, error) {
	// Try to load from .env file in current directory
	cfg, err := configuration.LoadConfig(".")
	if err != nil {
		log.Printf("Não foi possível carregar .env do diretório atual, tentando diretório pai...")
		// Try parent directory
		cfg, err = configuration.LoadConfig("..")
		if err != nil {
			log.Printf("Não foi possível carregar .env, usando apenas variáveis de ambiente")
			// Return configuration based only on environment variables
			return configuration.LoadConfig("/dev/null") // This will use only env vars
		}
	}
	return cfg, nil
}

// getRabbitMQURL gets RabbitMQ URL from environment or config
func getRabbitMQURL(cfg *configuration.Conf) string {
	rabbitmqURL := os.Getenv("ENV_RABBITMQ")
	if rabbitmqURL == "" {
		rabbitmqURL = cfg.ENV_RABBITMQ
	}
	return rabbitmqURL
}

// getHTTPPort gets HTTP port from environment or config
func getHTTPPort(cfg *configuration.Conf) string {
	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = cfg.HTTPPort
	}
	if httpPort == "" {
		httpPort = "8080" // Default port
	}
	return httpPort
}

// getWorkersCount gets the number of workers from environment or uses default
func getWorkersCount() int {
	workersStr := os.Getenv("WORKERS")
	if workersStr == "" {
		return 20 // Default number of workers
	}

	workers, err := strconv.Atoi(workersStr)
	if err != nil {
		log.Printf("Valor inválido para WORKERS: %s, usando padrão: 20", workersStr)
		return 20
	}

	if workers <= 0 {
		log.Printf("Número de workers deve ser positivo, usando padrão: 20")
		return 20
	}

	return workers
}

// maskRabbitMQURL masks sensitive information in the RabbitMQ URL for logging
func maskRabbitMQURL(url string) string {
	if len(url) > 20 {
		return url[:10] + "***" + url[len(url)-7:]
	}
	return "***"
}

// setupGracefulShutdown sets up graceful shutdown handling
func setupGracefulShutdown() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		sig := <-c
		log.Printf("Recebido sinal %v, iniciando shutdown graceful...", sig)

		// Here you could add cleanup logic if needed
		// For example, closing connections, finishing current work, etc.

		log.Println("Aplicação finalizada.")
		os.Exit(0)
	}()
}
