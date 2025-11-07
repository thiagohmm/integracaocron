package examples

import (
	"log"
	"os"
	"time"

	"github.com/thiagohmm/integracaocron/configuration"
	"github.com/thiagohmm/integracaocron/domain/entities"
	"github.com/thiagohmm/integracaocron/domain/repositories"
	"github.com/thiagohmm/integracaocron/domain/usecases"
	"github.com/thiagohmm/integracaocron/infraestructure/database"
)

// RunCompleteIntegrationExample shows how to use the PromotionUseCase with IntegrationJobUseCase
func RunCompleteIntegrationExample() {
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

	// Create all repositories
	promotionRepo := repositories.NewPromotionRepository(db)
	parameterRepo := repositories.NewParameterRepository(db)
	integrationRepo := repositories.NewIntegrationRepository(db)
	networkRepo := repositories.NewNetworkRepository(db)

	// Get RabbitMQ URL from environment or config
	rabbitmqURL := os.Getenv("ENV_RABBITMQ")
	if rabbitmqURL == "" {
		rabbitmqURL = cfg.ENV_RABBITMQ
	}

	// Create integration job use case
	integrationJobUC := usecases.NewIntegrationJobUseCase(parameterRepo, integrationRepo, networkRepo, db)

	// Create promotion use case with integration job
	promotionUC := usecases.NewPromotionUseCase(promotionRepo, rabbitmqURL, integrationJobUC)

	// Example promotion data
	testPromotion := entities.Promotion{
		IPMD_ID:         123,
		Json:            `{"descricao": "Promoção teste", "desconto": 10}`,
		DATARECEBIMENTO: time.Now().Format("2006-01-02 15:04:05"),
	}

	// Process promotion (this will call the integration job at the end)
	log.Println("Iniciando processamento de promoção...")
	err = promotionUC.ProcessIntegrationPromotions(testPromotion)
	if err != nil {
		log.Fatalf("Erro no processamento de promoção: %v", err)
	}

	log.Println("Processamento concluído com sucesso!")
	log.Println("A função produtNetworkMain foi chamada automaticamente no final do processamento.")
}
