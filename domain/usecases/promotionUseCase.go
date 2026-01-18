package usecases

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
	"github.com/thiagohmm/integracaocron/domain/entities"
	"github.com/thiagohmm/integracaocron/infraestructure/rabbitmq"
	"github.com/thiagohmm/integracaocron/pkg/tracing"
)

// PromotionUseCase handles promotion integration business logic
type PromotionUseCase struct {
	promotionRepo    entities.PromotionRepository
	rabbitmqURL      string
	integrationJobUC *IntegrationJobUseCase
}

// LogIntegrRMS represents the log structure for integration messages
type LogIntegrRMS struct {
	Tabela string        `json:"tabela"`
	Fields []string      `json:"fields"`
	Values []interface{} `json:"values"`
}

// NewPromotionUseCase creates a new instance of PromotionUseCase
func NewPromotionUseCase(promotionRepo entities.PromotionRepository, rabbitmqURL string, integrationJobUC *IntegrationJobUseCase) *PromotionUseCase {
	return &PromotionUseCase{
		promotionRepo:    promotionRepo,
		rabbitmqURL:      rabbitmqURL,
		integrationJobUC: integrationJobUC,
	}
}

// ProcessarPromocao processes promotion data from RabbitMQ message
// This method can be called from the listener
func (uc *PromotionUseCase) ProcessarPromocao(dados entities.Promotion) error {
	ctx := context.Background()
	ctx, span := tracing.StartSpan(ctx, "ProcessarPromocao")
	defer span.End()
	
	log.Printf("Iniciando processamento de promoção com dados: %+v", dados)

	// Validate promotion data
	if dados.IPM_ID == 0 {
		log.Printf("IPM_ID inválido (0), ignorando promoção vazia")
		tracing.SetStatus(ctx, 2, "IPM_ID inválido")
		return fmt.Errorf("IPM_ID inválido: não é possível processar promoção com ID 0")
	}

	// ✅ Adicionar idpromocao para pesquisa no Jaeger
	tracing.AddPromotionID(ctx, dados.IPM_ID)
	tracing.AddStringAttribute(ctx, "promotion.type", "individual")

	// Call the main integration processing
	return uc.ProcessIntegrationPromotions(dados)
}

// ProcessarTodasPromocoesPendentes fetches and processes all pending promotions from database
// This method should be called when receiving a simple "promocao" message
func (uc *PromotionUseCase) ProcessarTodasPromocoesPendentes() error {
	log.Printf("Iniciando processamento de todas as promoções pendentes...")

	// Fetch all pending promotions from database
	promotions, err := uc.promotionRepo.GetIntegrRMSPromocaoIN()
	if err != nil {
		log.Printf("Erro ao buscar promoções pendentes: %v", err)
		return fmt.Errorf("erro ao buscar promoções pendentes: %w", err)
	}

	if len(promotions) == 0 {
		log.Printf("Nenhuma promoção pendente encontrada para processar")
		return nil
	}

	log.Printf("Encontradas %d promoções pendentes para processar", len(promotions))

	// Process each promotion
	successCount := 0
	errorCount := 0

	for _, promo := range promotions {
		ctx := context.Background()
		ctx, span := tracing.StartSpan(ctx, "ProcessPromotionFromPending")
		defer span.End()
		
		// ✅ Adicionar idpromocao para pesquisa no Jaeger
		tracing.AddPromotionID(ctx, promo.IPM_ID)
		tracing.AddStringAttribute(ctx, "promotion.type", "pending")
		
		log.Printf("Processando promoção ID: %d", promo.IPM_ID)

		err := uc.ProcessIntegrationPromotions(promo)
		if err != nil {
			log.Printf("Erro ao processar promoção %d: %v", promo.IPM_ID, err)
			tracing.RecordError(ctx, err)
			errorCount++
		} else {
			log.Printf("Promoção %d processada com sucesso", promo.IPM_ID)
			successCount++
		}
	}

	log.Printf("Processamento concluído: %d sucessos, %d erros", successCount, errorCount)

	// Call the integration job at the end (optional - skip if tables don't exist)
	if uc.integrationJobUC != nil {
		log.Println("Chamando job de integração no final do processamento...")
		if err := uc.integrationJobUC.IntegrationJob(); err != nil {
			log.Printf("AVISO: Job de integração falhou (pode ser ignorado se tabelas não existem): %v", err)
			// Don't return error here to avoid failing the main promotion processing
		} else {
			log.Printf("Job de integração executado com sucesso")
		}
	}

	if errorCount > 0 {
		return fmt.Errorf("processamento concluído com %d erros de %d promoções", errorCount, len(promotions))
	}

	return nil
}

// ProcessIntegrationPromotions processes all pending promotion integrations
// This is the Go equivalent of the TypeScript function you provided
func (uc *PromotionUseCase) ProcessIntegrationPromotions(dados entities.Promotion) error {
	// Process the individual promotion
	uc.processIndividualPromotion(dados)

	// Call the integration job at the end (equivalent to productNetworkMain) - optional
	if uc.integrationJobUC != nil {
		log.Println("Chamando job de integração no final do processamento de promoção...")
		dataCorte := time.Now()
		if err := uc.integrationJobUC.ProductNetworkMain(dataCorte); err != nil {
			log.Printf("AVISO: Job de integração falhou (pode ser ignorado se tabelas não existem): %v", err)
			// Don't return error here to avoid failing the main promotion processing
			// The integration job error will be logged but won't affect the promotion result
		} else {
			log.Printf("Job de integração executado com sucesso")
		}
	}

	return nil
}

// processIndividualPromotion processes a single promotion with error handling
func (uc *PromotionUseCase) processIndividualPromotion(promo entities.Promotion) {
	ctx := context.Background()
	ctx, span := tracing.StartSpan(ctx, "processIndividualPromotion")
	defer span.End()
	
	// ✅ Adicionar idpromocao para pesquisa no Jaeger
	tracing.AddPromotionID(ctx, promo.IPM_ID)
	
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic while processing promotion %d: %v", promo.IPM_ID, r)
			tracing.AddEvent(ctx, "panic.recovered", tracing.StringAttr("panic_value", fmt.Sprintf("%v", r)))
			uc.handlePromotionError(promo, fmt.Errorf("panic: %v", r))
		}
	}()

	// Call the dopkg_promotion function (equivalent to the TypeScript version)
	promocao, err := uc.promotionRepo.Dopkg_promotion(promo.IPM_ID)
	if err != nil {
		log.Printf("Erro ao processar promoção %d: %v", promo.IPM_ID, err)
		uc.handlePromotionError(promo, err)
		return
	}

	log.Printf("promocao: %+v", promocao)

	// Delete the processed promotion (equivalent to deletePorObjeto)
	err = uc.deletePorObjeto(promo.IPM_ID)
	if err != nil {
		log.Printf("Erro ao deletar promoção %d: %v", promo.IPM_ID, err)
		// Continue processing and log the success/failure of the main operation
	}

	// Create success/failure log
	var statusProcessamento int
	var descricaoErro string

	if promocao.Success {
		statusProcessamento = 0
		descricaoErro = "Processamento realizado com sucesso."
	} else {
		statusProcessamento = 1
		descricaoErro = promocao.Message
	}

	// Convert promotion to JSON string
	promoJSON, _ := json.Marshal(promo)

	logSucesso := LogIntegrRMS{
		Tabela: "LogIntegrRMS",
		Fields: []string{"TRANSACAO", "TABELA", "DATARECEBIMENTO", "DATAPROCESSAMENTO", "STATUSPROCESSAMENTO", "JSON", "DESCRICAOERRO"},
		Values: []interface{}{
			"IN",
			"PROMOCAO",
			promo.DATARECEBIMENTO,
			time.Now().Format("2006-01-02 15:04:05"),
			statusProcessamento,
			string(promoJSON),
			descricaoErro,
		},
	}

	uc.sendToQueue(logSucesso)
}

// handlePromotionError handles errors that occur during promotion processing
func (uc *PromotionUseCase) handlePromotionError(promo entities.Promotion, err error) {
	log.Printf("Erro ao processar promoção: %v", err)

	// Delete the problematic promotion
	deleteErr := uc.deletePorObjeto(promo.IPM_ID)
	if deleteErr != nil {
		log.Printf("Erro ao deletar promoção com erro %d: %v", promo.IPM_ID, deleteErr)
	}

	// Convert prom otion to JSON string
	promoJSON, _ := json.Marshal(promo)

	// Create error log
	dataRecebimento := promo.DATARECEBIMENTO
	if dataRecebimento == "" {
		dataRecebimento = time.Now().Format("2006-01-02 15:04:05")
	}

	logErro := LogIntegrRMS{
		Tabela: "LogIntegrRMS",
		Fields: []string{"TRANSACAO", "TABELA", "DATARECEBIMENTO", "DATAPROCESSAMENTO", "STATUSPROCESSAMENTO", "JSON", "DESCRICAOERRO"},
		Values: []interface{}{
			"IN",
			"PROMOCAO",
			dataRecebimento,
			time.Now().Format("2006-01-02 15:04:05"),
			1,
			string(promoJSON),
			fmt.Sprintf("%v", err),
		},
	}

	log.Printf("Log de erro sendo enviado para a fila: %+v", logErro)
	uc.sendToQueue(logErro)
}

// deletePorObjeto deletes a promotion record by IPMD_ID
func (uc *PromotionUseCase) deletePorObjeto(ipmID int) error {
	return uc.promotionRepo.DeletePorObjeto(ipmID)
}

// sendToQueue sends a log message to RabbitMQ queue
func (uc *PromotionUseCase) sendToQueue(logData LogIntegrRMS) {
	// Get RabbitMQ connection
	conn, err := rabbitmq.GetRabbitMQConnection(uc.rabbitmqURL)
	if err != nil {
		log.Printf("Erro ao conectar ao RabbitMQ: %v", err)
		return
	}

	// Create channel
	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Erro ao criar canal RabbitMQ: %v", err)
		return
	}
	defer ch.Close()

	// Convert log data to JSON
	body, err := json.Marshal(logData)
	if err != nil {
		log.Printf("Erro ao converter log para JSON: %v", err)
		return
	}

	// Declare queue (ensure it exists)
	queueName := "log"
	_, err = ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		log.Printf("Erro ao declarar fila: %v", err)
		return
	}

	// Publish message
	err = ch.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "application/json",
			Body:        body,
		})

	if err != nil {
		log.Printf("Erro ao enviar mensagem para fila: %v", err)
	} else {
		log.Printf("Log enviado para fila com sucesso: %s", string(body))
	}
}
