package rabbitmq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/streadway/amqp"
	"github.com/thiagohmm/integracaocron/internal/infraestructure/cache"
	infraestructure "github.com/thiagohmm/integracaocron/internal/infraestructure/rabbitmq"
	"github.com/thiagohmm/integracaocron/internal/usecases"
	"go.opentelemetry.io/otel"
)

type Listener struct {
	EstruturaMercadologica  *usecases.EstruturaMercadologicaUseCase
	PromocaoUC              *usecases.PromocaoUseCase
	Produtos                 *usecases.ProdutosUseCase
	
	Workers                 int // número de workers concorrentes
}

func (l *Listener) getConnectionWithWait(rabbitmqurl string) (*amqp.Connection, error) {
	log.Printf("Iniciando tentativa de conexão com RabbitMQ...")

	for {
		conn, err := infraestructure.GetRabbitMQConnection(rabbitmqurl)
		if err == nil {
			log.Printf("Conexão com RabbitMQ estabelecida com sucesso")
			return conn, nil
		}
		log.Printf("Erro conectando ao RabbitMQ: %v. Tentando novamente em 5 segundos...", err)
		time.Sleep(5 * time.Second)
	}
}

func (l *Listener) ListenToQueue(rabbitmqurl string) error {
	if rabbitmqurl == "" {
		return fmt.Errorf("rabbitmq URL cannot be empty")
	}

	if l.Workers <= 0 {
		l.Workers = 20 // default to 20 workers if not set
	}

	log.Printf("Iniciando listener RabbitMQ com %d workers - Container sempre ativo", l.Workers)

	// Loop infinito para manter a aplicação sempre ativa
	for {
		log.Printf("Tentando conectar ao RabbitMQ...")

		conn, err := l.getConnectionWithWait(rabbitmqurl)
		if err != nil {
			log.Printf("Erro conectando ao RabbitMQ: %v. Tentando novamente em 5 segundos...", err)
			time.Sleep(5 * time.Second)
			continue
		}

		log.Printf("Conectado ao RabbitMQ com sucesso")

		ch, err := conn.Channel()
		if err != nil {
			log.Printf("Erro criando canal RabbitMQ: %v. Tentando reconectar...", err)
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		// Configurar prefetch count para controlar quantas mensagens cada worker recebe
		err = ch.Qos(
			l.Workers, // prefetch count
			0,         // prefetch size
			false,     // global
		)
		if err != nil {
			log.Printf("Erro configurando QoS: %v. Tentando reconectar...", err)
			ch.Close()
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		queue := "integracaoCron"
		msgs, err := ch.Consume(queue, "", false, false, false, false, nil)
		if err != nil {
			log.Printf("Erro consumindo mensagens: %v. Tentando reconectar...", err)
			ch.Close()
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		log.Printf("Canal de mensagens criado com sucesso para a fila: %s", queue)

		// WaitGroup para controlar os workers
		var wg sync.WaitGroup

		// Canal para sinalizar shutdown dos workers
		//workerShutdown := make(chan struct{})

		// Iniciar workers
		for i := 0; i < l.Workers; i++ {
			wg.Add(1)
			//go l.worker(i, msgs, &wg, workerShutdown)
			go l.worker(i, msgs, &wg, nil) // Passando nil para workerShutdown, pois não estamos usando shutdown neste exemplo
		}

		log.Printf("Listener iniciado com %d workers - Aguardando mensagens...", l.Workers)

		// Monitorar a conexão para detectar quando ela fecha
		connClosed := make(chan *amqp.Error, 1)
		conn.NotifyClose(connClosed)

		// Aguardar até que a conexão seja fechada
		select {
		case err := <-connClosed:
			if err != nil {
				log.Printf("Conexão RabbitMQ fechada com erro: %v. Reiniciando workers...", err)
			} else {
				log.Printf("Conexão RabbitMQ fechada normalmente. Reiniciando workers...")
			}
		}

		// Sinalizar para os workers pararem
		//close(workerShutdown)

		// Fechar canal de mensagens para parar os workers
		ch.Close()

		// Aguardar todos os workers terminarem
		log.Printf("Aguardando workers terminarem...")
		wg.Wait()
		log.Printf("Todos os workers finalizados. Reconectando em 5 segundos...")

		// Fechar conexão
		conn.Close()

		// Pequena pausa antes de tentar reconectar
		time.Sleep(5 * time.Second)
	}

	// Este ponto nunca deve ser alcançado
	return nil
}

func restartApplication() {
	exe, err := os.Executable()
	if err != nil {
		log.Fatalf("Erro ao obter executável para reiniciar: %v", err)
	}
	log.Printf("Reiniciando aplicação: %s", exe)
	if err := syscall.Exec(exe, os.Args, os.Environ()); err != nil {
		log.Fatalf("Erro ao reiniciar aplicação: %v", err)
	}
}

func (l *Listener) worker(id int, msgs <-chan amqp.Delivery, wg *sync.WaitGroup, workerShutdown <-chan struct{}) {
	defer wg.Done()
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Worker %d recovered from panic: %v", id, r)
			restartApplication()
		}
	}()

	log.Printf("Worker %d iniciado e aguardando mensagens...", id)
	
	

	messageCount := 0
	idleTime := time.Now()

	for msg := range msgs {
		// Log de tempo ocioso se passou muito tempo
		if time.Since(idleTime) > 30*time.Second {
			log.Printf("Worker %d - Primeira mensagem após %v de ociosidade", id, time.Since(idleTime))
		}

		messageCount++
		log.Printf("Worker %d processando mensagem #%d", id, messageCount)

		if err, _ := l.processMessage(msg); err != nil {
			// Criar span para rastreamento de erro
		

			log.Printf("Worker %d - Erro processando mensagem #%d: %v", id, messageCount, err)

		

		
		} else {
			log.Printf("Worker %d - Mensagem #%d processada com sucesso", id, messageCount)
		}

		if err := msg.Ack(false); err != nil {
			log.Printf("Worker %d - Erro ao confirmar mensagem #%d: %v. Tentando enviar Nack...", id, messageCount, err)
			// Se falhar o ack, enviar nack sem requeue para não tentar processar novamente
			if nackErr := msg.Nack(false, false); nackErr != nil {
				log.Printf("Worker %d - Erro ao enviar Nack para a mensagem #%d: %v", id, messageCount, nackErr)
			} else {
				log.Printf("Worker %d - Nack enviado com sucesso para a mensagem #%d", id, messageCount)
			}
		} else {
			log.Printf("Worker %d - Mensagem #%d confirmada com sucesso", id, messageCount)
		}
		// Resetar tempo ocioso
		idleTime = time.Now()
	}

}

func (l *Listener) processMessage(msg amqp.Delivery) (error, string) {
	log.Printf("Iniciando processamento de mensagem...")

	var message map[string]interface{}
	if err := json.Unmarshal(msg.Body, &message); err != nil {
		log.Printf("Erro ao fazer parse da mensagem: %v", err)
		return fmt.Errorf("erro ao fazer parse da mensagem: %w", err), ""
	}

	tipoIntegracao, ok := message["tipoIntegracao"].(string)
	if !ok {
		log.Printf("Campo 'tipoIntegracao' inválido ou ausente na mensagem")
		return fmt.Errorf("campo 'tipoIntegracao' inválido ou ausente"), ""
	}

	

	dados, ok := message["dados"].(map[string]interface{})
	if !ok {
		log.Printf("Campo 'dados' inválido ou ausente na mensagem para UUID: %s", uuid)
		return fmt.Errorf("campo 'dados' inválido ou ausente"), uuid
	}

	
	var err error

	switch tipoIntegracao {
	case "EstruturaMercadologica":
		log.Printf("Iniciando processamento de Estrutura mercadológica: %s")
		_, err = l.EstruturaMercadologica.ProcessarEstrutura( dados)
		log.Printf("Processamento de estrutura mercadológica concluído para UUID: %s")
	case "Promocao":
		log.Printf("Iniciando processamento de promoção: %s")
		err = l.PromocaoUC.ProcessarPromocao( dados)
		log.Printf("Processamento de promoção concluído: %s")
	case "Produtos":
		log.Printf("Iniciando processamento de produtos: %s")
		err = l.EstoqueUC.ProcessarProdutos( dados)
		log.Printf("Processamento de produtos concluído: %s")
	default:
		log.Printf("Tipo de processo desconhecido: %s", tipoIntegracao)
		return fmt.Errorf("tipo de processo desconhecido: %s", tipoIntegracao), ""
	}

	if err != nil {
		log.Printf("Erro processando mensagem do tipo '%s' com UUID '%s': %v", tipoIntegracao, uuid, err)
		
		return err
	}

	
	return nil,
}
