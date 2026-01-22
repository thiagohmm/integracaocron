package repositories

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/streadway/amqp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"github.com/thiagohmm/integracaocron/domain/entities"
	"github.com/thiagohmm/integracaocron/infraestructure/rabbitmq"
)

// IntegrationRepositoryImpl implements the IntegrationRepository interface
type IntegrationRepositoryImpl struct {
	db          *sql.DB
	rabbitmqURL string
}

// NewIntegrationRepository creates a new instance of IntegrationRepository
func NewIntegrationRepository(db *sql.DB) entities.IntegrationRepository {
	return &IntegrationRepositoryImpl{
		db: db,
	}
}

// SetRabbitMQURL sets the RabbitMQ URL for sending messages
func (r *IntegrationRepositoryImpl) SetRabbitMQURL(url string) {
	r.rabbitmqURL = url
}

// generateUUID generates a UUID v4
func generateUUID() string {
	uuid := make([]byte, 16)
	rand.Read(uuid)
	// Set version (4) and variant bits
	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		hex.EncodeToString(uuid[0:4]),
		hex.EncodeToString(uuid[4:6]),
		hex.EncodeToString(uuid[6:8]),
		hex.EncodeToString(uuid[8:10]),
		hex.EncodeToString(uuid[10:16]))
}

// SendToQueue sends a message to RabbitMQ queue
func (r *IntegrationRepositoryImpl) SendToQueue(message entities.QueueMessage) error {
	if r.rabbitmqURL == "" {
		// Fallback: log if RabbitMQ URL is not configured
		messageJSON, _ := json.Marshal(message)
		log.Printf("RabbitMQ URL not configured, logging message: %s", string(messageJSON))
		return nil
	}

	// Get RabbitMQ connection
	conn, err := rabbitmq.GetRabbitMQConnection(r.rabbitmqURL)
	if err != nil {
		log.Printf("Erro ao conectar ao RabbitMQ: %v", err)
		return err
	}

	// Create channel
	ch, err := conn.Channel()
	if err != nil {
		log.Printf("Erro ao criar canal RabbitMQ: %v", err)
		return err
	}
	defer ch.Close()

	// Convert message to JSON
	body, err := json.Marshal(message)
	if err != nil {
		log.Printf("Erro ao converter mensagem para JSON: %v", err)
		return err
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
		return err
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
		return err
	}

	return nil
}

// Transaction removal methods
func (r *IntegrationRepositoryImpl) RemoveIntegrationCombo(dataCorte time.Time, expurgo ...string) error {
	// Timeout aumentado para 600 segundos (10 minutos) pois a stored procedure
	// pode processar grandes volumes de dados e demorar mais tempo
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	// Set default value for expurgo if not provided
	fazExpurgo := "NAO"
	if len(expurgo) > 0 {
		fazExpurgo = expurgo[0]
	}

	query := `BEGIN sp_limparintegracaocombocorte(:1, :2); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte, fazExpurgo)
	if err != nil {
		// Verificar se é erro de timeout/cancelamento
		if strings.Contains(err.Error(), "ORA-01013") || 
		   strings.Contains(err.Error(), "user requested cancel") ||
		   strings.Contains(err.Error(), "context deadline exceeded") {
			log.Printf("Erro de timeout ao remover integração combo (stored procedure demorou mais de 10 minutos): %v", err)
			return fmt.Errorf("timeout ao remover integração combo: a operação demorou mais de 10 minutos. Considere verificar o volume de dados ou otimizar a stored procedure: %w", err)
		}
		log.Printf("Erro ao executar sp_limparintegracaocombocorte: %v", err)
		return fmt.Errorf("erro ao remover integração combo: %w", err)
	}

	log.Printf("Integração combo removida com sucesso para data corte: %v, expurgo: %s", dataCorte, fazExpurgo)
	return nil
}

func (r *IntegrationRepositoryImpl) ClearIntegrationPackagingByCutOffDate(dataCorte time.Time, expurgo ...string) error {
	// Timeout aumentado para 600 segundos (10 minutos) pois a stored procedure
	// pode processar grandes volumes de dados e demorar mais tempo
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	// Set default value
	fazExpurgo := "NAO"
	if len(expurgo) > 0 {
		fazExpurgo = expurgo[0]
	}

	query := `BEGIN sp_LimparIntegracaoEmbalagemCorte(:1, :2); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte, fazExpurgo)
	if err != nil {
		// Verificar se é erro de timeout/cancelamento
		if strings.Contains(err.Error(), "ORA-01013") || 
		   strings.Contains(err.Error(), "user requested cancel") ||
		   strings.Contains(err.Error(), "context deadline exceeded") {
			log.Printf("Erro de timeout ao limpar integração embalagem (stored procedure demorou mais de 10 minutos): %v", err)
			return fmt.Errorf("timeout ao limpar integração embalagem: a operação demorou mais de 10 minutos. Considere verificar o volume de dados ou otimizar a stored procedure: %w", err)
		}
		log.Printf("Erro ao limpar integração embalagem: %v", err)
		return fmt.Errorf("erro ao limpar integração embalagem: %w", err)
	}

	return nil
}

func (r *IntegrationRepositoryImpl) RemoverTransacaoIntegracaoEstruturaMercadologica(dataCorte time.Time, expurgo ...string) error {
	// Timeout aumentado para 600 segundos (10 minutos) pois a stored procedure
	// pode processar grandes volumes de dados e demorar mais tempo
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	// Set default value
	fazExpurgo := "NAO"
	if len(expurgo) > 0 {
		fazExpurgo = expurgo[0]
	}

	query := `BEGIN sp_limparintegracaoestruturamercadologicacorte(:1, :2); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte, fazExpurgo)
	if err != nil {
		// Verificar se é erro de timeout/cancelamento
		if strings.Contains(err.Error(), "ORA-01013") || 
		   strings.Contains(err.Error(), "user requested cancel") ||
		   strings.Contains(err.Error(), "context deadline exceeded") {
			log.Printf("Erro de timeout ao remover transação estrutura mercadológica (stored procedure demorou mais de 10 minutos): %v", err)
			return fmt.Errorf("timeout ao remover transação estrutura mercadológica: a operação demorou mais de 10 minutos. Considere verificar o volume de dados ou otimizar a stored procedure: %w", err)
		}
		log.Printf("Erro ao remover transação estrutura mercadológica: %v", err)
		return fmt.Errorf("erro ao remover transação estrutura mercadológica: %w", err)
	}

	return nil
}

func (r *IntegrationRepositoryImpl) RemoverTransacaoIntegracaoProduto(dataCorte time.Time, expurgo ...string) error {
	// Timeout aumentado para 600 segundos (10 minutos) pois a stored procedure
	// pode processar grandes volumes de dados e demorar mais tempo
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	// Set default value
	fazExpurgo := "NAO"
	if len(expurgo) > 0 {
		fazExpurgo = expurgo[0]
	}

	query := `BEGIN sp_limparintegracaoprodutocorte(:1, :2); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte, fazExpurgo)
	if err != nil {
		// Verificar se é erro de timeout/cancelamento
		if strings.Contains(err.Error(), "ORA-01013") || 
		   strings.Contains(err.Error(), "user requested cancel") ||
		   strings.Contains(err.Error(), "context deadline exceeded") {
			log.Printf("Erro de timeout ao remover transação produto (stored procedure demorou mais de 10 minutos): %v", err)
			return fmt.Errorf("timeout ao remover transação produto: a operação demorou mais de 10 minutos. Considere verificar o volume de dados ou otimizar a stored procedure: %w", err)
		}
		log.Printf("Erro ao remover transação produto: %v", err)
		return fmt.Errorf("erro ao remover transação produto: %w", err)
	}

	return nil
}

func (r *IntegrationRepositoryImpl) RemoverTransacaoIntegracaoPromocao(dataCorte time.Time, expurgo ...string) error {
	// Timeout aumentado para 600 segundos (10 minutos) pois a stored procedure
	// pode processar grandes volumes de dados e demorar mais tempo
	ctx, cancel := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancel()

	// Set default value
	fazExpurgo := "NAO"
	if len(expurgo) > 0 {
		fazExpurgo = expurgo[0]
	}

	query := `BEGIN sp_limparintegracaopromocaocorte(:1, :2); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte, fazExpurgo)
	if err != nil {
		// Verificar se é erro de timeout/cancelamento
		if strings.Contains(err.Error(), "ORA-01013") || 
		   strings.Contains(err.Error(), "user requested cancel") ||
		   strings.Contains(err.Error(), "context deadline exceeded") {
			log.Printf("Erro de timeout ao remover transação promoção (stored procedure demorou mais de 10 minutos): %v", err)
			return fmt.Errorf("timeout ao remover transação promoção: a operação demorou mais de 10 minutos. Considere verificar o volume de dados ou otimizar a stored procedure: %w", err)
		}
		log.Printf("Erro ao remover transação promoção: %v", err)
		return fmt.Errorf("erro ao remover transação promoção: %w", err)
	}

	return nil
}

func (r *IntegrationRepositoryImpl) CheckMarketingStructure() (bool, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `SELECT COUNT(*) FROM INTEGRACAOESTRUTURAMERCADOLOGICA WHERE ENVIANDO = '1'`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		log.Printf("Erro ao verificar estrutura de marketing: %v", err)
		return false, fmt.Errorf("erro ao verificar estrutura de marketing: %w", err)
	}

	return count > 0, nil
}

func (r *IntegrationRepositoryImpl) CheckProductIntegration() (bool, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `SELECT COUNT(*) FROM INTEGRACAOPRODUTO WHERE ENVIANDO = '1'`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		log.Printf("Erro ao verificar integração de produto: %v", err)
		return false, fmt.Errorf("erro ao verificar integração de produto: %w", err)
	}

	return count > 0, nil
}

func (r *IntegrationRepositoryImpl) CheckPackagingIntegration() (bool, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT COUNT(*) FROM INTEGRACAOEMBALAGEM WHERE ENVIANDO = '1'`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		log.Printf("Erro ao verificar integração de embalagem: %v", err)
		return false, fmt.Errorf("erro ao verificar integração de embalagem: %w", err)
	}

	return count > 0, nil
}

func (r *IntegrationRepositoryImpl) CheckComboIntegration() (bool, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT COUNT(*) FROM INTEGRACAOCOMBO WHERE ENVIANDO = '1'`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		log.Printf("Erro ao verificar integração de combo: %v", err)
		return false, fmt.Errorf("erro ao verificar integração de combo: %w", err)
	}

	return count > 0, nil
}

func (r *IntegrationRepositoryImpl) CheckPromotionIntegration() (bool, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT COUNT(*) FROM INTEGRACAOPROMOCAO WHERE ENVIANDO = '1'`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		log.Printf("Erro ao verificar integração de promoção: %v", err)
		return false, fmt.Errorf("erro ao verificar integração de promoção: %w", err)
	}

	return count > 0, nil
}

// Check Staging Tables methods
func (r *IntegrationRepositoryImpl) HasMarketingStructureStaging() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT COUNT(*) FROM INTEGRACAOESTRUTURAMERCADOLOGICASTAGING`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		log.Printf("Erro ao verificar staging estrutura mercadológica: %v", err)
		return false, fmt.Errorf("erro ao verificar staging estrutura mercadológica: %w", err)
	}

	return count > 0, nil
}

func (r *IntegrationRepositoryImpl) HasProductStaging() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT COUNT(*) FROM INTEGRACAOPRODUTOSTAGING`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		log.Printf("Erro ao verificar staging produto: %v", err)
		return false, fmt.Errorf("erro ao verificar staging produto: %w", err)
	}

	return count > 0, nil
}

func (r *IntegrationRepositoryImpl) HasPackagingStaging() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT COUNT(*) FROM INTEGRACAOEMBALAGEMSTAGING`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		log.Printf("Erro ao verificar staging embalagem: %v", err)
		return false, fmt.Errorf("erro ao verificar staging embalagem: %w", err)
	}

	return count > 0, nil
}

func (r *IntegrationRepositoryImpl) HasComboStaging() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT COUNT(*) FROM INTEGRACAOCOMBOSTAGING`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		log.Printf("Erro ao verificar staging combo: %v", err)
		return false, fmt.Errorf("erro ao verificar staging combo: %w", err)
	}

	return count > 0, nil
}

func (r *IntegrationRepositoryImpl) HasPromotionStaging() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT COUNT(*) FROM INTEGRACAOPROMOCAOSTAGING`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		log.Printf("Erro ao verificar staging promoção: %v", err)
		return false, fmt.Errorf("erro ao verificar staging promoção: %w", err)
	}

	return count > 0, nil
}

// Data movement methods
func (r *IntegrationRepositoryImpl) MoveIntegrationMarketingStructure(dataCorte time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	query := `BEGIN sp_MoverStagingEstruturaMercadologica(:1); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte)
	if err != nil {
		log.Printf("Erro ao executar sp_MoverStagingEstruturaMercadologica: %v", err)
		return fmt.Errorf("erro ao mover estrutura mercadológica para staging: %w", err)
	}

	log.Printf("Estrutura mercadológica movida para staging com sucesso para data: %v", dataCorte)
	return nil
}

func (r *IntegrationRepositoryImpl) MoveIntegrationProductStaging(ctx context.Context, dataCorte time.Time) (string, error) {
	// Gerar UUID para esta transação
	transactionUUID := generateUUID()

	// Adicionar UUID ao contexto do Jaeger
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetAttributes(
			attribute.String("transaction_uuid", transactionUUID),
			attribute.String("transaction_type", "mover_produto"),
			attribute.String("uuid", transactionUUID), // Adicionar também como "uuid" para facilitar busca
		)
		// Adicionar evento com UUID para facilitar busca no Jaeger
		span.AddEvent("transaction.started", trace.WithAttributes(
			attribute.String("transaction_uuid", transactionUUID),
			attribute.String("transaction_type", "mover_produto"),
		))
	}

	// Criar contexto com timeout para a query
	queryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	query := `BEGIN sp_MoverStagingProduto(:1); END;`

	_, err := r.db.ExecContext(queryCtx, query, dataCorte)
	if err != nil {
		log.Printf("Erro ao executar sp_MoverStagingProduto: %v", err)
		return transactionUUID, fmt.Errorf("erro ao mover produto para staging: %w", err)
	}

	log.Printf("Produto movido para staging com sucesso para data: %v, UUID: %s", dataCorte, transactionUUID)

	// Enviar UUID para a fila log
	logMessage := entities.QueueMessage{
		Tabela: "LogsIntegrRMS",
		Fields: []string{"TRANSACAO", "TABELA", "DATARECEBIMENTO", "DATAPROCESSAMENTO", "STATUSPROCESSAMENTO", "JSON", "DESCRICAOERRO"},
		Values: []interface{}{
			transactionUUID,
			"PRODUTOS",
			dataCorte,
			time.Now(),
			0, // 0 = sucesso
			fmt.Sprintf(`{"transaction_uuid":"%s","transaction_type":"mover_produto","data_corte":"%s"}`, transactionUUID, dataCorte.Format(time.RFC3339)),
			fmt.Sprintf("Produto movido para staging com sucesso - UUID: %s", transactionUUID),
		},
	}

	if err := r.SendToQueue(logMessage); err != nil {
		log.Printf("Erro ao enviar UUID para fila log: %v", err)
		// Não retornar erro aqui, apenas logar
	}

	return transactionUUID, nil
}

func (r *IntegrationRepositoryImpl) MoveIntegrationPackagingStaging(dataCorte time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	query := `BEGIN sp_MoverStagingEmbalagem(:1); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte)
	if err != nil {
		log.Printf("Erro ao executar sp_MoverStagingEmbalagem: %v", err)
		return fmt.Errorf("erro ao mover embalagem para staging: %w", err)
	}

	log.Printf("Embalagem movida para staging com sucesso para data: %v", dataCorte)
	return nil
}

func (r *IntegrationRepositoryImpl) MoveIntegrationComboStaging(dataCorte time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	query := `BEGIN sp_MoverStagingCombo(:1); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte)
	if err != nil {
		log.Printf("Erro ao executar sp_MoverStagingCombo: %v", err)
		return fmt.Errorf("erro ao mover combo para staging: %w", err)
	}

	log.Printf("Combo movido para staging com sucesso para data: %v", dataCorte)
	return nil
}

func (r *IntegrationRepositoryImpl) MoveIntegrationPromotionStaging(ctx context.Context, dataCorte time.Time) (string, error) {
	// Gerar UUID para esta transação
	transactionUUID := generateUUID()

	// Adicionar UUID ao contexto do Jaeger
	span := trace.SpanFromContext(ctx)
	if span != nil {
		span.SetAttributes(
			attribute.String("transaction_uuid", transactionUUID),
			attribute.String("transaction_type", "mover_promocao"),
			attribute.String("uuid", transactionUUID), // Adicionar também como "uuid" para facilitar busca
		)
		// Adicionar evento com UUID para facilitar busca no Jaeger
		span.AddEvent("transaction.started", trace.WithAttributes(
			attribute.String("transaction_uuid", transactionUUID),
			attribute.String("transaction_type", "mover_promocao"),
		))
	}

	// Criar contexto com timeout para a query
	queryCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	query := `BEGIN sp_MoverStagingPromocao(:1); END;`

	_, err := r.db.ExecContext(queryCtx, query, dataCorte)
	if err != nil {
		log.Printf("Erro ao executar sp_MoverStagingPromocao: %v", err)
		return transactionUUID, fmt.Errorf("erro ao mover promoção para staging: %w", err)
	}

	log.Printf("Promoção movida para staging com sucesso para data: %v, UUID: %s", dataCorte, transactionUUID)

	// Enviar UUID para a fila log
	logMessage := entities.QueueMessage{
		Tabela: "LogsIntegrRMS",
		Fields: []string{"TRANSACAO", "TABELA", "DATARECEBIMENTO", "DATAPROCESSAMENTO", "STATUSPROCESSAMENTO", "JSON", "DESCRICAOERRO"},
		Values: []interface{}{
			transactionUUID,
			"PROMOCOES",
			dataCorte,
			time.Now(),
			0, // 0 = sucesso
			fmt.Sprintf(`{"transaction_uuid":"%s","transaction_type":"mover_promocao","data_corte":"%s"}`, transactionUUID, dataCorte.Format(time.RFC3339)),
			fmt.Sprintf("Promoção movida para staging com sucesso - UUID: %s", transactionUUID),
		},
	}

	if err := r.SendToQueue(logMessage); err != nil {
		log.Printf("Erro ao enviar UUID para fila log: %v", err)
		// Não retornar erro aqui, apenas logar
	}

	return transactionUUID, nil
}

// Expiry methods
func (r *IntegrationRepositoryImpl) GetIntegrationUpdateComboByDate(dataCorte time.Time) ([]entities.IntegrationCombo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	query := `
		SELECT IDINTEGRACAOCOMBO, IDREVENDEDOR, IDCOMBOPROMOCAO, 
			   ENVIANDO, JSON, DATAATUALIZACAO, TRANSACAO, DATAINICIOENVIO
		FROM INTEGRACAOCOMBO WHERE DATAATUALIZACAO < :1`

	rows, err := r.db.QueryContext(ctx, query, dataCorte)
	if err != nil {
		log.Printf("Erro ao consultar combos para expurgo: %v", err)
		return nil, fmt.Errorf("erro ao consultar combos: %w", err)
	}
	defer rows.Close()

	var combos []entities.IntegrationCombo
	for rows.Next() {
		var combo entities.IntegrationCombo
		err := rows.Scan(
			&combo.IdIntegracaoCombo,
			&combo.IdRevendedor,
			&combo.IdComboPromocao,
			&combo.Enviando,
			&combo.Json,
			&combo.DataAtualizacao,
			&combo.Transacao,
			&combo.DataInicioEnvio,
		)
		if err != nil {
			log.Printf("Erro ao escanear combo: %v", err)
			continue
		}
		combos = append(combos, combo)
	}

	return combos, nil
}

func (r *IntegrationRepositoryImpl) DeleteIntegrationCombo(idIntegracaoCombo int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `DELETE FROM INTEGRACAOCOMBO WHERE IDINTEGRACAOCOMBO = :1`

	_, err := r.db.ExecContext(ctx, query, idIntegracaoCombo)
	if err != nil {
		log.Printf("Erro ao deletar combo %d: %v", idIntegracaoCombo, err)
		return fmt.Errorf("erro ao deletar combo: %w", err)
	}

	return nil
}

func (r *IntegrationRepositoryImpl) UpdateExpiredSlaSolicitation() error {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	query := `BEGIN sp_AtualizarVencimentoSlaSolicitacoes(); END;`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		log.Printf("Erro ao executar sp_AtualizarVencimentoSlaSolicitacoes: %v", err)
		return fmt.Errorf("erro ao atualizar vencimento SLA solicitações: %w", err)
	}

	log.Printf("Vencimento SLA das solicitações atualizado com sucesso")
	return nil
}
