package rabbitmq

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"syscall"
	"time"

	"github.com/streadway/amqp"
	"github.com/thiagohmm/integracaocron/domain/entities"
	"github.com/thiagohmm/integracaocron/domain/usecases"
	infraestructure "github.com/thiagohmm/integracaocron/infraestructure/rabbitmq"
)

type Listener struct {
	PromocaoUC               *usecases.PromotionUseCase
	IntegrationUc            *usecases.IntegrationJobUseCase
	ProductIntegrationUC     *usecases.ProductIntegrationUseCase
	PromotionNormalizationUC *usecases.PromotionNormalizationUseCase

	//EstruturaMercadologica *usecases.EstruturaMercadologicaUseCase --- IGNORE ---
	//Produtos               *usecases.ProdutosUseCase --- IGNORE ---

	Workers int // número de workers concorrentes
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

		// Declare queue to ensure it exists
		_, err = ch.QueueDeclare(
			queue, // name
			true,  // durable
			false, // delete when unused
			false, // exclusive
			false, // no-wait
			nil,   // arguments
		)
		if err != nil {
			log.Printf("Erro declarando fila %s: %v. Tentando reconectar...", queue, err)
			ch.Close()
			conn.Close()
			time.Sleep(5 * time.Second)
			continue
		}

		log.Printf("Fila '%s' declarada com sucesso", queue)

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
		closeErr := <-connClosed
		if closeErr != nil {
			log.Printf("Conexão RabbitMQ fechada com erro: %v. Reiniciando workers...", closeErr)
		} else {
			log.Printf("Conexão RabbitMQ fechada normalmente. Reiniciando workers...")
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
	log.Printf("Mensagem recebida (raw): %s", string(msg.Body))

	var tipoIntegracao string
	var dados map[string]interface{}

	// Primeiro, tenta fazer parse como JSON object
	var message map[string]interface{}
	if err := json.Unmarshal(msg.Body, &message); err == nil {
		// É um JSON object válido
		log.Printf("Mensagem parseada como JSON object")

		// Verifica se é o formato novo com "type_message"
		if typeMsg, ok := message["type_message"].(string); ok {
			tipoIntegracao = typeMsg
			// Dados podem estar em "dados" ou a mensagem inteira pode ser os dados
			if d, ok := message["dados"].(map[string]interface{}); ok {
				dados = d
			} else {
				dados = message
			}
		} else if tipoInt, ok := message["tipoIntegracao"].(string); ok {
			// Formato antigo com "tipoIntegracao"
			tipoIntegracao = tipoInt
			if d, ok := message["dados"].(map[string]interface{}); ok {
				dados = d
			} else {
				dados = message
			}
		} else {
			log.Printf("Campo 'type_message' ou 'tipoIntegracao' não encontrado no JSON object")
			return fmt.Errorf("campo 'type_message' ou 'tipoIntegracao' não encontrado no JSON object"), ""
		}
	} else {
		// Se falhou o parse como JSON object, tenta como string JSON
		var simpleMessage string
		if err := json.Unmarshal(msg.Body, &simpleMessage); err == nil {
			// É uma string JSON válida
			log.Printf("Mensagem parseada como string JSON: %s", simpleMessage)
			tipoIntegracao = simpleMessage
			dados = make(map[string]interface{})
		} else {
			// Se não é JSON válido, trata como string simples
			log.Printf("Mensagem não é JSON válido, tratando como string simples")
			messageStr := string(msg.Body)
			// Remove aspas simples se existirem
			if len(messageStr) >= 2 && messageStr[0] == '\'' && messageStr[len(messageStr)-1] == '\'' {
				messageStr = messageStr[1 : len(messageStr)-1]
			}
			// Remove aspas duplas se existirem
			if len(messageStr) >= 2 && messageStr[0] == '"' && messageStr[len(messageStr)-1] == '"' {
				messageStr = messageStr[1 : len(messageStr)-1]
			}
			tipoIntegracao = messageStr
			dados = make(map[string]interface{})
			log.Printf("Tipo de integração extraído da string simples: %s", tipoIntegracao)
		}
	}

	log.Printf("Tipo de integração detectado: %s", tipoIntegracao)

	switch tipoIntegracao {

	case "promocao", "Promocao":
		log.Printf("Iniciando processamento de promoção")
		var promocao entities.Promotion
		promocaoBytes, err := json.Marshal(dados)
		if err != nil {
			log.Printf("Erro ao serializar dados de promoção: %v", err)
			return fmt.Errorf("erro ao serializar dados de promoção: %w", err), ""
		}
		if err := json.Unmarshal(promocaoBytes, &promocao); err != nil {
			log.Printf("Erro ao desserializar dados para entities.Promotion: %v", err)
			return fmt.Errorf("erro ao desserializar dados para entities.Promotion: %w", err), ""
		}
		err = l.PromocaoUC.ProcessarPromocao(promocao)
		if err != nil {
			log.Printf("Erro ao processar promoção: %v", err)
			return fmt.Errorf("erro ao processar promoção: %w", err), ""
		}
		err = l.IntegrationUc.IntegrationJob()
		if err != nil {
			log.Printf("Erro ao processar integração: %v", err)
			return fmt.Errorf("erro ao processar integração: %w", err), ""
		}

		log.Printf("Processamento de promoção concluído")

	case "produto", "Produto":
		log.Printf("Iniciando processamento de produto")

		if l.ProductIntegrationUC == nil {
			log.Printf("ProductIntegrationUC não foi inicializado")
			return fmt.Errorf("ProductIntegrationUC não foi inicializado"), ""
		}

		success, err := l.ProductIntegrationUC.ImportProductIntegration()
		if err != nil {
			log.Printf("Erro ao processar integração de produtos: %v", err)
			return fmt.Errorf("erro ao processar integração de produtos: %w", err), ""
		}

		if !success {
			log.Printf("Integração de produtos concluída com alguns erros")
			return fmt.Errorf("integração de produtos concluída com alguns erros"), ""
		}

		log.Printf("Processamento de produto concluído com sucesso")

	case "promocao_normalizacao", "PromocaoNormalizacao":
		log.Printf("Iniciando normalização de promoções")

		if l.PromotionNormalizationUC == nil {
			log.Printf("PromotionNormalizationUC não foi inicializado")
			return fmt.Errorf("PromotionNormalizationUC não foi inicializado"), ""
		}

		if l.IntegrationUc == nil {
			log.Printf("IntegrationUc não foi inicializado")
			return fmt.Errorf("IntegrationUc não foi inicializado"), ""
		}

		result, err := l.PromotionNormalizationUC.NormalizePromotions()
		if err != nil {
			log.Printf("Erro ao processar normalização de promoções: %v", err)
			return fmt.Errorf("erro ao processar normalização de promoções: %w", err), ""
		}

		if !result.Success {
			log.Printf("Normalização de promoções concluída com alguns erros: %s", result.Message)
			return fmt.Errorf("normalização de promoções concluída com alguns erros: %s", result.Message), ""
		}

		log.Printf("Normalização de promoções concluída com sucesso. Processados: %d, Atualizados: %d, Duplicatas removidas: %d",
			result.ProcessedCount, result.UpdatedCount, result.TotalRemovedDuplicates)

	case "mover", "productNetworkMain", "product_network_main":
		log.Printf("Iniciando processo ProductNetworkMain")

		if l.IntegrationUc == nil {
			log.Printf("IntegrationUc não foi inicializado")
			return fmt.Errorf("IntegrationUc não foi inicializado"), ""
		}

		// Usar time.Now() como dataCorte
		dataCorte := time.Now()

		err := l.productNetworkMain(dataCorte)
		if err != nil {
			log.Printf("Erro ao executar ProductNetworkMain: %v", err)
			return fmt.Errorf("erro ao executar ProductNetworkMain: %w", err), ""
		}

		log.Printf("Processo ProductNetworkMain concluído com sucesso")

	default:
		log.Printf("Tipo de processo desconhecido: %s", tipoIntegracao)
		return fmt.Errorf("tipo de processo desconhecido: %s", tipoIntegracao), ""
	}

	return nil, ""
}

// productNetworkMain executa o job principal de integração de produtos e rede
// Baseado na função TypeScript productNetworkMain
func (l *Listener) productNetworkMain(dataCorte time.Time) error {
	log.Printf("Job Integração - Início")

	// Executar integração principal
	if err := l.IntegrationUc.IntegrationJob(); err != nil {
		log.Printf("Erro ao executar integração: %v", err)
		return fmt.Errorf("erro ao executar integração: %w", err)
	}

	// Executar job de replicação de produtos de rede
	if err := l.replicateNetworkProductsJob(); err != nil {
		log.Printf("Erro ao replicar produtos de rede: %v", err)
		return fmt.Errorf("erro ao replicar produtos de rede: %w", err)
	}

	// Mover dados usando o dataCorte fornecido
	if err := l.IntegrationUc.MoveDataJob(dataCorte); err != nil {
		log.Printf("Erro ao mover dados: %v", err)
		return fmt.Errorf("erro ao mover dados: %w", err)
	}

	// Atualizar solicitações SLA expiradas
	if err := l.IntegrationUc.UpdateExpirationSlaRequestsJob(); err != nil {
		log.Printf("Erro ao atualizar solicitações SLA expiradas: %v", err)
		return fmt.Errorf("erro ao atualizar solicitações SLA expiradas: %w", err)
	}

	log.Printf("Job Integração - Término")
	return nil
}

// replicateNetworkProductsJob replica produtos para as redes
// Baseado na função TypeScript replicateNetworkProductsJob
func (l *Listener) replicateNetworkProductsJob() error {
	log.Printf("Replicar produtos redes - Início.")

	// Por enquanto, apenas um log pois precisaríamos implementar:
	// - NetworkQuery para buscar redes
	// - DealerNetworkQuery para buscar lojas
	// - Lógica de replicação de produtos
	// Esta implementação pode ser expandida conforme necessário

	log.Printf("Replicar produtos redes - Fim.")
	return nil
}
