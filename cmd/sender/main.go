package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

	"github.com/streadway/amqp"
	"github.com/thiagohmm/integracaocron/configuration"
)

func main() {
	var messageType = flag.String("type", "promocao", "Type of message: promocao, produto, produto_integracao, promocao_normalizacao, mover")
	var message = flag.String("msg", "", "Custom message content (JSON)")
	flag.Parse()

	// Load configuration
	cfg, err := configuration.LoadConfig(".env")
	if err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
	}

	// Connect to RabbitMQ
	conn, err := amqp.Dial(cfg.ENV_RABBITMQ)
	if err != nil {
		log.Fatalf("Failed to connect to RabbitMQ: %v", err)
	}
	defer conn.Close()

	ch, err := conn.Channel()
	if err != nil {
		log.Fatalf("Failed to open a channel: %v", err)
	}
	defer ch.Close()

	queueName := "integracaoCron"

	// Declare queue to ensure it exists
	_, err = ch.QueueDeclare(
		queueName, // name
		true,      // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		log.Fatalf("Failed to declare queue: %v", err)
	}

	var messageBody string

	if *message != "" {
		messageBody = *message
	} else {
		// Create sample messages based on type
		switch *messageType {
		case "promocao":
			messageBody = "promocao"
		case "produto":
			messageBody = "produto"
		case "produto_integracao":
			sampleMessage := map[string]interface{}{
				"type_message": "produto_integracao",
				"data": map[string]interface{}{
					"codigo":    "12345",
					"descricao": "Produto de teste",
					"categoria": "Categoria A",
					"preco":     99.99,
					"ativo":     true,
				},
			}
			msgBytes, _ := json.Marshal(sampleMessage)
			messageBody = string(msgBytes)
		case "promocao_normalizacao":
			sampleMessage := map[string]interface{}{
				"type_message": "promocao_normalizacao",
				"data": map[string]interface{}{
					"codigo":      "PROMO001",
					"descricao":   "Promoção de teste",
					"desconto":    15.0,
					"data_inicio": "2025-01-01",
					"data_fim":    "2025-01-31",
				},
			}
			msgBytes, _ := json.Marshal(sampleMessage)
			messageBody = string(msgBytes)
		case "mover", "productNetworkMain", "product_network_main":
			messageBody = "mover"
		default:
			log.Fatalf("Unknown message type: %s", *messageType)
		}
	}

	// Send message
	err = ch.Publish(
		"",        // exchange
		queueName, // routing key
		false,     // mandatory
		false,     // immediate
		amqp.Publishing{
			ContentType: "text/plain",
			Body:        []byte(messageBody),
		},
	)
	if err != nil {
		log.Fatalf("Failed to publish message: %v", err)
	}

	fmt.Printf("✅ Message sent successfully to queue '%s':\n%s\n", queueName, messageBody)
}
