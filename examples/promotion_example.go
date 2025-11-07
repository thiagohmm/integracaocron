package examples

import (
	"log"
	"os"

	"github.com/thiagohmm/integracaocron/configuration"
	"github.com/thiagohmm/integracaocron/domain/entities"
	"github.com/thiagohmm/integracaocron/domain/repositories"
	"github.com/thiagohmm/integracaocron/domain/usecases"
	"github.com/thiagohmm/integracaocron/infraestructure/database"
)

// RunPromotionExample shows how to use the PromotionUseCase with IntegrationJobUseCase
func RunPromotionExample() {
	// Load configuration
	cfg, err := configuration.LoadConfig("../.env")
	if err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
	}

	// Connect to database
	db, err := database.ConectarBanco(cfg)
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco: %v", err)
	}
	defer db.Close()

	// Create repositories (Note: You'll need to implement these)
	promotionRepo := repositories.NewPromotionRepository(db)
	// parameterRepo := repositories.NewParameterRepository(db)
	// integrationRepo := repositories.NewIntegrationRepository(db)
	// networkRepo := repositories.NewNetworkRepository(db)

	// Get RabbitMQ URL from environment or config
	rabbitmqURL := os.Getenv("ENV_RABBITMQ")
	if rabbitmqURL == "" {
		rabbitmqURL = cfg.ENV_RABBITMQ
	}

	// Create integration job use case (uncomment when repositories are implemented)
	// integrationJobUC := usecases.NewIntegrationJobUseCase(parameterRepo, integrationRepo, networkRepo, db)

	// Create promotion use case with integration job
	// promotionUC := usecases.NewPromotionUseCase(promotionRepo, rabbitmqURL, integrationJobUC)

	// For now, create without integration job (pass nil)
	promotionUC := usecases.NewPromotionUseCase(promotionRepo, rabbitmqURL, nil)

	// Example promotion data
	testPromotion := entities.Promotion{
		IPMD_ID:         123,
		Json:            `{"test": "data"}`,
		DATARECEBIMENTO: "2025-10-06 12:00:00",
	}

	// Process promotion
	log.Println("Iniciando processamento de promoção...")
	err = promotionUC.ProcessIntegrationPromotions(testPromotion)
	if err != nil {
		log.Fatalf("Erro no processamento de promoção: %v", err)
	}

	log.Println("Processamento concluído com sucesso!")
}
