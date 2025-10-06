package rabbitmq

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/streadway/amqp"
	"github.com/thiagohmm/integracaocron/configuration"
)

var (
	globalConn *amqp.Connection
	connMux    sync.Mutex
)

// RabbitMQConsumer define a estrutura para consumir mensagens via RabbitMQ.
type RabbitMQConsumer struct {
	URL string
	// Outras propriedades se necessário...
}

// HealthChecker define o contrato de saúde para o RabbitMQConsumer.
type HealthChecker interface {
	IsHealthy() bool
}

// GetRabbitMQConnection retorna a conexão ativa com o RabbitMQ, reutilizando-a se ela já estiver estabelecida.
func GetRabbitMQConnection(rabbitmqURL string) (*amqp.Connection, error) {
	if rabbitmqURL == "" {
		return nil, fmt.Errorf("RABBITMQ_URL is not defined")
	}

	connMux.Lock()
	defer connMux.Unlock()

	// Se já existe uma conexão ativa, retorna-a:
	if globalConn != nil && !globalConn.IsClosed() {
		return globalConn, nil
	}

	// Tentativa de conexão em loop até obter sucesso.
	for {
		conn, err := amqp.Dial(rabbitmqURL)
		if err == nil {
			globalConn = conn
			log.Println("Successfully connected to RabbitMQ")
			// Inicia o monitoramento da conexão em background.
			go monitorConnection(rabbitmqURL)
			return globalConn, nil
		}
		log.Printf("Failed to connect to RabbitMQ: %v. Retrying in 5 seconds...", err)
		time.Sleep(5 * time.Second)
	}
}

// monitorConnection monitora a conexão e reconecta caso a conexão seja fechada.
func monitorConnection(rabbitmqURL string) {
	for {
		time.Sleep(1 * time.Second)
		connMux.Lock()
		if globalConn == nil || globalConn.IsClosed() {
			connMux.Unlock()
			// Reproduz o mesmo loop de conexão até obter sucesso.
			for {
				conn, err := amqp.Dial(rabbitmqURL)
				if err == nil {
					connMux.Lock()
					globalConn = conn
					connMux.Unlock()
					log.Println("Successfully reconnected to RabbitMQ")
					break
				}
				log.Printf("Failed to reconnect to RabbitMQ: %v. Retrying in 5 seconds...", err)
				time.Sleep(5 * time.Second)
			}
		} else {
			connMux.Unlock()
		}
	}
}

// IsHealthy tenta reutilizar a conexão e retorna true se ela estiver ativa.
func (c RabbitMQConsumer) IsHealthy() bool {
	rabbitmqURL := os.Getenv("ENV_RABBITMQ")
	if rabbitmqURL == "" {
		cfg, err := loadConfig()
		if err != nil {
			log.Printf("Failed to load config: %v", err)
			return false
		}
		rabbitmqURL = cfg.ENV_RABBITMQ
	}
	conn, err := GetRabbitMQConnection(rabbitmqURL)
	if err != nil {
		log.Printf("RabbitMQ connection failed: %v", err)
		return false
	}
	// A conexão é reutilizada. Não fechar aqui.
	return conn != nil && !conn.IsClosed()
}

func loadConfig() (*configuration.Conf, error) {
	cfg, err := configuration.LoadConfig("../../.env")
	if err != nil {
		log.Fatalf("Erro ao carregar configuração: %v", err)
	}
	return cfg, err
}
