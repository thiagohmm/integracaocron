package main

import (
	"database/sql"
	"log"
	"os"

	"github.com/thiagohmm/integracaocron/domain/repositories"
	"github.com/thiagohmm/integracaocron/domain/usecases"
	rabbitmq "github.com/thiagohmm/integracaocron/internal/delivery"
)

// runPromotionNormalizationService demonstrates how to set up the promotion normalization service
func runPromotionNormalizationService() {
	log.Println("Starting Promotion Normalization Service...")

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
	promotionNormalizationRepo := repositories.NewPromotionNormalizationRepository(db)

	// Initialize use cases
	promotionNormalizationUC := usecases.NewPromotionNormalizationUseCase(promotionNormalizationRepo, db)

	// For complete setup, you would also initialize:
	// Other use cases as needed

	// Get RabbitMQ URL from environment or use default
	rabbitmqURL := os.Getenv("RABBITMQ_URL")
	if rabbitmqURL == "" {
		rabbitmqURL = "amqp://guest:guest@localhost:5672/"
	}

	// Initialize listener with promotion normalization support
	listener := &rabbitmq.Listener{
		// PromocaoUC:                promotionUC,               // Initialize when needed
		// IntegrationUc:             integrationJobUC,          // Initialize when needed
		// ProductIntegrationUC:      productIntegrationUC,     // Initialize when needed
		PromotionNormalizationUC: promotionNormalizationUC,
		Workers:                  20, // Number of concurrent workers
	}

	log.Println("Promotion Normalization Service initialized successfully")
	log.Println("Starting RabbitMQ listener...")

	// Start listening to RabbitMQ (this will run indefinitely)
	// NOTE: You need to implement the actual database connection first
	_ = listener
	_ = rabbitmqURL
	log.Println("Example function - implement database connection to use")
}

// testPromotionNormalization demonstrates how to manually test promotion normalization
func testPromotionNormalization() {
	log.Println("Testing Promotion Normalization...")

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
	promotionNormalizationRepo := repositories.NewPromotionNormalizationRepository(db)
	promotionNormalizationUC := usecases.NewPromotionNormalizationUseCase(promotionNormalizationRepo, db)

	// Run promotion normalization
	result, err := promotionNormalizationUC.NormalizePromotions()
	if err != nil {
		log.Printf("Error during promotion normalization: %v", err)
		return
	}

	if result.Success {
		log.Printf("Promotion normalization completed successfully")
		log.Printf("Processed: %d, Updated: %d, Duplicates Removed: %d",
			result.ProcessedCount, result.UpdatedCount, result.TotalRemovedDuplicates)
	} else {
		log.Printf("Promotion normalization completed with errors: %s", result.Message)
	}
}

// Example RabbitMQ message for promotion normalization
/*
Message Format for Promotion Normalization:
{
  "tipoIntegracao": "PromocaoNormalizacao",
  "dados": {
    // any additional filtering parameters if needed
  }
}

The actual promotion data is read from the INTEGRACAO_PROMOCAO table,
not from the RabbitMQ message itself. The message just triggers the normalization process.

What the normalization does:
1. Reads all records from INTEGRACAO_PROMOCAO table
2. For each record, parses the JSON field containing promotion data
3. For each group (grupos) in the promotion:
   - Removes duplicate items based on codBarra (barcode)
   - Updates qtdeItem to reflect the new count
4. Updates the record with normalized JSON
5. Logs all changes to the queue for monitoring

Example of JSON structure being normalized:
{
  "codMix": "12345",
  "grupos": [
    {
      "desc": "Group 1",
      "qtdeItem": 5,
      "items": [
        {"codBarra": "123", "desc": "Product 1", "preco": 10.0, "qtde": 1},
        {"codBarra": "123", "desc": "Product 1", "preco": 10.0, "qtde": 1},  // Duplicate - will be removed
        {"codBarra": "456", "desc": "Product 2", "preco": 20.0, "qtde": 2}
      ]
    }
  ]
}

After normalization, the duplicate item with codBarra "123" is removed,
and qtdeItem is updated from 5 to 2.
*/
