package usecases

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/streadway/amqp"
	"github.com/thiagohmm/integracaocron/domain/entities"
	"github.com/thiagohmm/integracaocron/infraestructure/rabbitmq"
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
	log.Printf("Iniciando processamento de promoção com dados: %+v", dados)

	// Call the main integration processing
	return uc.ProcessIntegrationPromotions(dados)
}

// ProcessIntegrationPromotions processes all pending promotion integrations
// This is the Go equivalent of the TypeScript function you provided
func (uc *PromotionUseCase) ProcessIntegrationPromotions(dados entities.Promotion) error {
	// Process the individual promotion
	uc.processIndividualPromotion(dados)

	// Call the integration job at the end (equivalent to productNetworkMain)
	if uc.integrationJobUC != nil {
		log.Println("Chamando job de integração no final do processamento de promoção...")
		dataCorte := time.Now()
		if err := uc.integrationJobUC.ProductNetworkMain(dataCorte); err != nil {
			log.Printf("Erro ao executar job de integração: %v", err)
			// Don't return error here to avoid failing the main promotion processing
			// The integration job error will be logged but won't affect the promotion result
		}
	}

	return nil
}

// processIndividualPromotion processes a single promotion with error handling
func (uc *PromotionUseCase) processIndividualPromotion(promo entities.Promotion) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic while processing promotion %d: %v", promo.IPMD_ID, r)
			uc.handlePromotionError(promo, fmt.Errorf("panic: %v", r))
		}
	}()

	// Call the dopkg_promotion function (equivalent to the TypeScript version)
	promocao, err := uc.promotionRepo.Dopkg_promotion(promo.IPMD_ID)
	if err != nil {
		log.Printf("Erro ao processar promoção %d: %v", promo.IPMD_ID, err)
		uc.handlePromotionError(promo, err)
		return
	}

	log.Printf("promocao: %+v", promocao)

	// Delete the processed promotion (equivalent to deletePorObjeto)
	err = uc.deletePorObjeto(promo.IPMD_ID)
	if err != nil {
		log.Printf("Erro ao deletar promoção %d: %v", promo.IPMD_ID, err)
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
	deleteErr := uc.deletePorObjeto(promo.IPMD_ID)
	if deleteErr != nil {
		log.Printf("Erro ao deletar promoção com erro %d: %v", promo.IPMD_ID, deleteErr)
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
