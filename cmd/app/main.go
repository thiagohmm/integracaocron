package main

import (
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/thiagohmm/integracaocron/configuration"
	"github.com/thiagohmm/integracaocron/domain/repositories"
	"github.com/thiagohmm/integracaocron/domain/usecases"
	"github.com/thiagohmm/integracaocron/infraestructure/database"
	rabbitmq "github.com/thiagohmm/integracaocron/internal/delivery"
)

func main() {
	log.Println("=== Iniciando aplicação IntegracaoCron ===")

	// Load configuration
	cfg, err := loadConfiguration()
	if err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
	}

	// Connect to database
	db, err := database.ConectarBanco(cfg)
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco de dados: %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			log.Printf("Erro ao fechar conexão com o banco: %v", err)
		}
	}()

	// Initialize repositories
	promotionRepo := repositories.NewPromotionRepository(db)
	parameterRepo := repositories.NewParameterRepository(db)
	integrationRepo := repositories.NewIntegrationRepository(db)
	networkRepo := repositories.NewNetworkRepository(db)

	// Get RabbitMQ configuration
	rabbitmqURL := getRabbitMQURL(cfg)
	if rabbitmqURL == "" {
		log.Fatal("URL do RabbitMQ não configurada")
	}

	// Initialize use cases
	integrationJobUC := usecases.NewIntegrationJobUseCase(parameterRepo, integrationRepo, networkRepo, db)
	promotionUC := usecases.NewPromotionUseCase(promotionRepo, rabbitmqURL, integrationJobUC)

	// Get number of workers from environment or use default
	workers := getWorkersCount()

	// Initialize RabbitMQ listener
	listener := &rabbitmq.Listener{
		PromocaoUC: promotionUC,
		Workers:    workers,
	}

	// Setup graceful shutdown
	setupGracefulShutdown()

	// Start listening to RabbitMQ
	log.Printf("Iniciando listener RabbitMQ com %d workers", workers)
	log.Printf("Conectando ao RabbitMQ: %s", maskRabbitMQURL(rabbitmqURL))

	if err := listener.ListenToQueue(rabbitmqURL); err != nil {
		log.Fatalf("Erro ao iniciar listener RabbitMQ: %v", err)
	}
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
