package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/thiagohmm/integracaocron/domain/repositories"
	"github.com/thiagohmm/integracaocron/domain/usecases"
	rabbitmq "github.com/thiagohmm/integracaocron/internal/delivery"
)

func runProductIntegrationService() {
	log.Println("Starting Product Integration Service...")

	// Initialize database connection (you'll need to implement this based on your config)
	// cfg := &config.Conf{} // Load your config
	// db, err := database.ConectarBanco(cfg)
	// if err != nil {
	// 	log.Fatalf("Failed to connect to database: %v", err)
	// }
	// defer db.Close()

	// For example purposes, using nil - replace with actual database connection
	var db *sql.DB = nil

	// Initialize repositories
	productIntegrationRepo := repositories.NewProductIntegrationRepository(db)

	// Initialize use cases
	productIntegrationUC := usecases.NewProductIntegrationUseCase(productIntegrationRepo, db)

	// For complete setup, you would also initialize:
	// parameterRepo := repositories.NewParameterRepository(db)
	// integrationRepo := repositories.NewIntegrationRepository(db)
	// networkRepo := repositories.NewNetworkRepository(db)
	// promotionRepo := repositories.NewPromotionRepository(db)
	// integrationJobUC := usecases.NewIntegrationJobUseCase(parameterRepo, integrationRepo, networkRepo, db)
	// promotionUC := usecases.NewPromotionUseCase(promotionRepo, "connectionString", integrationJobUC)

	// Get RabbitMQ URL from environment or use default
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://guest:guest@localhost:5672/"
	}

	// Initialize listener with product integration support
	listener := &rabbitmq.Listener{
		// PromocaoUC:           promotionUC,     // Initialize when needed
		// IntegrationUc:        integrationJobUC, // Initialize when needed
		ProductIntegrationUC: productIntegrationUC,
		Workers:              20, // Number of concurrent workers
	}

	log.Println("Product Integration Service initialized successfully")
	log.Println("Starting RabbitMQ listener...")

	// Start listening to RabbitMQ (this will run indefinitely)
	// NOTE: You need to implement the actual database connection first
	_ = listener
	_ = rabbitmqURL
	log.Println("Example function - implement database connection to use")
}

// Example of how to test product integration manually
func testProductIntegration() {
	log.Println("Testing Product Integration...")

	// Initialize database connection
	// cfg := &config.Conf{} // Load your config
	// db, err := database.ConectarBanco(cfg)
	// if err != nil {
	// 	log.Fatalf("Failed to connect to database: %v", err)
	// }
	// defer db.Close()

	// For example purposes, using nil - replace with actual database connection
	var db *sql.DB = nil

	// Initialize repository and use case
	productIntegrationRepo := repositories.NewProductIntegrationRepository(db)
	productIntegrationUC := usecases.NewProductIntegrationUseCase(productIntegrationRepo, db)

	// Run product integration
	success, err := productIntegrationUC.ImportProductIntegration()
	if err != nil {
		log.Printf("Error during product integration: %v", err)
		return
	}

	if success {
		log.Println("Product integration completed successfully")
	} else {
		log.Println("Product integration completed with some errors")
	}
}

// Example RabbitMQ message for product integration
/*
Message Format for Product Integration:
{
  "tipoIntegracao": "Produto",
  "dados": {
    "any additional data if needed for specific product filtering"
  }
}

The actual product data is read from the INTEGR_RMS_PRODUTO_IN table,
not from the RabbitMQ message itself. The message just triggers the integration process.
*/
