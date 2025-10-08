package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/thiagohmm/integracaocron/domain/entities"
)

// IntegrationPackagingRepositoryImpl implements the IntegrationPackagingRepository interface
type IntegrationPackagingRepositoryImpl struct {
	db *sql.DB
}

// NewIntegrationPackagingRepository creates a new instance of IntegrationPackagingRepository
func NewIntegrationPackagingRepository(db *sql.DB) entities.IntegrationPackagingRepository {
	return &IntegrationPackagingRepositoryImpl{
		db: db,
	}
}

// GetIntegrationUpdateByDate retrieves integrations by update date
func (r *IntegrationPackagingRepositoryImpl) GetIntegrationUpdateByDate(date time.Time) ([]entities.IntegrationPackaging, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		SELECT ID_INTEGRACAO_EMBALAGEM, ID_REVENDEDOR, ID_EMBALAGEM_PRODUTO, 
			   ENVIANDO, JSON, DATA_ATUALIZACAO, TRANSACAO, DATA_INICIO_ENVIO
		FROM INTEGRACAO_EMBALAGEM 
		WHERE DATA_ATUALIZACAO <= :1`

	rows, err := r.db.QueryContext(ctx, query, date)
	if err != nil {
		log.Printf("Erro ao consultar integrações de embalagem por data de atualização: %v", err)
		return nil, fmt.Errorf("erro ao consultar integrações: %w", err)
	}
	defer rows.Close()

	var integrations []entities.IntegrationPackaging
	for rows.Next() {
		var integration entities.IntegrationPackaging
		err := rows.Scan(
			&integration.IdIntegracaoEmbalagem,
			&integration.IdRevendedor,
			&integration.IdEmbalagemProduto,
			&integration.Enviando,
			&integration.Json,
			&integration.DataAtualizacao,
			&integration.Transacao,
			&integration.DataInicioEnvio,
		)
		if err != nil {
			log.Printf("Erro ao escanear integração de embalagem: %v", err)
			continue
		}
		integrations = append(integrations, integration)
	}

	log.Printf("Encontradas %d integrações de embalagem para a data %v", len(integrations), date)
	return integrations, nil
}

// GetIntegrationsByCodIbm retrieves integrations by IBM code and transaction ID
func (r *IntegrationPackagingRepositoryImpl) GetIntegrationsByCodIbm(codigoIbm string, transactionId string) ([]entities.IntegrationPackaging, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// First get dealer by IBM code
	dealerQuery := `SELECT ID_REVENDEDOR FROM REVENDEDOR WHERE CODIGO_IBM = :1`

	var dealerId int
	err := r.db.QueryRowContext(ctx, dealerQuery, codigoIbm).Scan(&dealerId)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Revendedor não encontrado para código IBM: %s", codigoIbm)
			return []entities.IntegrationPackaging{}, nil // Return empty slice instead of error
		}
		log.Printf("Erro ao consultar revendedor por código IBM %s: %v", codigoIbm, err)
		return nil, fmt.Errorf("erro ao consultar revendedor: %w", err)
	}

	// Now get integrations for this dealer
	query := `
		SELECT ID_INTEGRACAO_EMBALAGEM, ID_REVENDEDOR, ID_EMBALAGEM_PRODUTO, 
			   ENVIANDO, JSON, DATA_ATUALIZACAO, TRANSACAO, DATA_INICIO_ENVIO
		FROM INTEGRACAO_EMBALAGEM 
		WHERE ID_REVENDEDOR = :1 
		  AND ENVIANDO = '0' 
		  AND TRANSACAO = :2`

	rows, err := r.db.QueryContext(ctx, query, dealerId, transactionId)
	if err != nil {
		log.Printf("Erro ao consultar integrações de embalagem por código IBM: %v", err)
		return nil, fmt.Errorf("erro ao consultar integrações: %w", err)
	}
	defer rows.Close()

	var integrations []entities.IntegrationPackaging
	for rows.Next() {
		var integration entities.IntegrationPackaging
		err := rows.Scan(
			&integration.IdIntegracaoEmbalagem,
			&integration.IdRevendedor,
			&integration.IdEmbalagemProduto,
			&integration.Enviando,
			&integration.Json,
			&integration.DataAtualizacao,
			&integration.Transacao,
			&integration.DataInicioEnvio,
		)
		if err != nil {
			log.Printf("Erro ao escanear integração de embalagem: %v", err)
			continue
		}
		integrations = append(integrations, integration)
	}

	log.Printf("Encontradas %d integrações de embalagem para código IBM %s e transação %s", len(integrations), codigoIbm, transactionId)
	return integrations, nil
}

// GetIntegrations retrieves integrations excluding a specific one
func (r *IntegrationPackagingRepositoryImpl) GetIntegrations(ip *entities.IntegrationPackaging) ([]entities.IntegrationPackaging, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		SELECT ID_INTEGRACAO_EMBALAGEM, ID_REVENDEDOR, ID_EMBALAGEM_PRODUTO, 
			   ENVIANDO, JSON, DATA_ATUALIZACAO, TRANSACAO, DATA_INICIO_ENVIO
		FROM INTEGRACAO_EMBALAGEM 
		WHERE ID_INTEGRACAO_EMBALAGEM != :1
		  AND ID_REVENDEDOR = :2
		  AND ID_EMBALAGEM_PRODUTO = :3
		  AND ENVIANDO = '0'`

	rows, err := r.db.QueryContext(ctx, query, ip.IdIntegracaoEmbalagem, ip.IdRevendedor, ip.IdEmbalagemProduto)
	if err != nil {
		log.Printf("Erro ao consultar integrações de embalagem relacionadas: %v", err)
		return nil, fmt.Errorf("erro ao consultar integrações relacionadas: %w", err)
	}
	defer rows.Close()

	var integrations []entities.IntegrationPackaging
	for rows.Next() {
		var integration entities.IntegrationPackaging
		err := rows.Scan(
			&integration.IdIntegracaoEmbalagem,
			&integration.IdRevendedor,
			&integration.IdEmbalagemProduto,
			&integration.Enviando,
			&integration.Json,
			&integration.DataAtualizacao,
			&integration.Transacao,
			&integration.DataInicioEnvio,
		)
		if err != nil {
			log.Printf("Erro ao escanear integração de embalagem: %v", err)
			continue
		}
		integrations = append(integrations, integration)
	}

	log.Printf("Encontradas %d integrações de embalagem relacionadas", len(integrations))
	return integrations, nil
}

// GetTransactionByRemove retrieves integrations being sent by date
func (r *IntegrationPackagingRepositoryImpl) GetTransactionByRemove(date time.Time) ([]entities.IntegrationPackaging, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		SELECT ID_INTEGRACAO_EMBALAGEM, ID_REVENDEDOR, ID_EMBALAGEM_PRODUTO, 
			   ENVIANDO, JSON, DATA_ATUALIZACAO, TRANSACAO, DATA_INICIO_ENVIO
		FROM INTEGRACAO_EMBALAGEM 
		WHERE ENVIANDO = '1' 
		  AND DATA_INICIO_ENVIO <= :1`

	rows, err := r.db.QueryContext(ctx, query, date)
	if err != nil {
		log.Printf("Erro ao consultar transações de embalagem para remoção por data: %v", err)
		return nil, fmt.Errorf("erro ao consultar transações: %w", err)
	}
	defer rows.Close()

	var integrations []entities.IntegrationPackaging
	for rows.Next() {
		var integration entities.IntegrationPackaging
		err := rows.Scan(
			&integration.IdIntegracaoEmbalagem,
			&integration.IdRevendedor,
			&integration.IdEmbalagemProduto,
			&integration.Enviando,
			&integration.Json,
			&integration.DataAtualizacao,
			&integration.Transacao,
			&integration.DataInicioEnvio,
		)
		if err != nil {
			log.Printf("Erro ao escanear integração de embalagem: %v", err)
			continue
		}
		integrations = append(integrations, integration)
	}

	log.Printf("Encontradas %d transações de embalagem para remoção na data %v", len(integrations), date)
	return integrations, nil
}

// RemoveById removes an integration by ID
func (r *IntegrationPackagingRepositoryImpl) RemoveById(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `DELETE FROM INTEGRACAO_EMBALAGEM WHERE ID_INTEGRACAO_EMBALAGEM = :1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		log.Printf("Erro ao remover integração de embalagem %d: %v", id, err)
		return fmt.Errorf("erro ao remover integração: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Erro ao verificar linhas afetadas: %v", err)
		return fmt.Errorf("erro ao verificar remoção: %w", err)
	}

	if rowsAffected == 0 {
		log.Printf("Nenhuma integração de embalagem foi removida para ID: %d", id)
		return fmt.Errorf("integração não encontrada para remoção: %d", id)
	}

	log.Printf("Integração de embalagem %d removida com sucesso", id)
	return nil
}

// MoveIntegrationPackagingStaging moves integrations to staging
func (r *IntegrationPackagingRepositoryImpl) MoveIntegrationPackagingStaging(dataCorte time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

// ClearIntegrationPackagingByDealer clears integrations by dealer
func (r *IntegrationPackagingRepositoryImpl) ClearIntegrationPackagingByDealer(idDealer int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `BEGIN sp_LimparIntegracaoEmbalagemRevendedor(:1); END;`

	_, err := r.db.ExecContext(ctx, query, idDealer)
	if err != nil {
		log.Printf("Erro ao executar sp_LimparIntegracaoEmbalagemRevendedor: %v", err)
		return fmt.Errorf("erro ao limpar integração de embalagem por revendedor: %w", err)
	}

	log.Printf("Integração de embalagem limpa com sucesso para revendedor: %d", idDealer)
	return nil
}

// ClearIntegrationPackagingByCutOffDate clears integrations by cutoff date
func (r *IntegrationPackagingRepositoryImpl) ClearIntegrationPackagingByCutOffDate(cutOffDate time.Time, doPurge string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if doPurge == "" {
		doPurge = "NAO"
	}

	query := `BEGIN sp_LimparIntegracaoEmbalagemCorte(:1, :2); END;`

	_, err := r.db.ExecContext(ctx, query, cutOffDate, doPurge)
	if err != nil {
		log.Printf("Erro ao executar sp_LimparIntegracaoEmbalagemCorte: %v", err)
		return fmt.Errorf("erro ao limpar integração de embalagem por data de corte: %w", err)
	}

	log.Printf("Integração de embalagem limpa com sucesso para data: %v, expurgo: %s", cutOffDate, doPurge)
	return nil
}
