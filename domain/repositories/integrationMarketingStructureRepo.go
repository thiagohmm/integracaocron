package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/thiagohmm/integracaocron/domain/entities"
)

// IntegrationMarketingStructureRepositoryImpl implements the IntegrationMarketingStructureRepository interface
type IntegrationMarketingStructureRepositoryImpl struct {
	db *sql.DB
}

// NewIntegrationMarketingStructureRepository creates a new instance of IntegrationMarketingStructureRepository
func NewIntegrationMarketingStructureRepository(db *sql.DB) entities.IntegrationMarketingStructureRepository {
	return &IntegrationMarketingStructureRepositoryImpl{
		db: db,
	}
}

// GetIntegrationUpdateByDate retrieves integrations by update date
func (r *IntegrationMarketingStructureRepositoryImpl) GetIntegrationUpdateByDate(date time.Time) ([]entities.IntegrationMarketingStructure, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		SELECT ID_INTEGRACAO_ESTRUTURA_MERCADOLOGICA, ID_REVENDEDOR, ID_ESTRUTURA_MERCADOLOGICA, 
			   ENVIANDO, JSON, DATA_ATUALIZACAO, TRANSACAO, DATA_INICIO_ENVIO
		FROM INTEGRACAO_ESTRUTURA_MERCADOLOGICA 
		WHERE DATA_ATUALIZACAO <= :1`

	rows, err := r.db.QueryContext(ctx, query, date)
	if err != nil {
		log.Printf("Erro ao consultar integrações por data de atualização: %v", err)
		return nil, fmt.Errorf("erro ao consultar integrações: %w", err)
	}
	defer rows.Close()

	var integrations []entities.IntegrationMarketingStructure
	for rows.Next() {
		var integration entities.IntegrationMarketingStructure
		err := rows.Scan(
			&integration.IdIntegracaoEstruturaMercadologica,
			&integration.IdRevendedor,
			&integration.IdEstruturaMercadologica,
			&integration.Enviando,
			&integration.Json,
			&integration.DataAtualizacao,
			&integration.Transacao,
			&integration.DataInicioEnvio,
		)
		if err != nil {
			log.Printf("Erro ao escanear integração: %v", err)
			continue
		}
		integrations = append(integrations, integration)
	}

	log.Printf("Encontradas %d integrações para a data %v", len(integrations), date)
	return integrations, nil
}

// GetIntegrations retrieves integrations excluding a specific one
func (r *IntegrationMarketingStructureRepositoryImpl) GetIntegrations(ip *entities.IntegrationMarketingStructure) ([]entities.IntegrationMarketingStructure, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		SELECT ID_INTEGRACAO_ESTRUTURA_MERCADOLOGICA, ID_REVENDEDOR, ID_ESTRUTURA_MERCADOLOGICA, 
			   ENVIANDO, JSON, DATA_ATUALIZACAO, TRANSACAO, DATA_INICIO_ENVIO
		FROM INTEGRACAO_ESTRUTURA_MERCADOLOGICA 
		WHERE ID_INTEGRACAO_ESTRUTURA_MERCADOLOGICA != :1
		  AND ID_REVENDEDOR = :2
		  AND ID_ESTRUTURA_MERCADOLOGICA = :3
		  AND ENVIANDO = '0'`

	rows, err := r.db.QueryContext(ctx, query, ip.IdIntegracaoEstruturaMercadologica, ip.IdRevendedor, ip.IdEstruturaMercadologica)
	if err != nil {
		log.Printf("Erro ao consultar integrações relacionadas: %v", err)
		return nil, fmt.Errorf("erro ao consultar integrações relacionadas: %w", err)
	}
	defer rows.Close()

	var integrations []entities.IntegrationMarketingStructure
	for rows.Next() {
		var integration entities.IntegrationMarketingStructure
		err := rows.Scan(
			&integration.IdIntegracaoEstruturaMercadologica,
			&integration.IdRevendedor,
			&integration.IdEstruturaMercadologica,
			&integration.Enviando,
			&integration.Json,
			&integration.DataAtualizacao,
			&integration.Transacao,
			&integration.DataInicioEnvio,
		)
		if err != nil {
			log.Printf("Erro ao escanear integração: %v", err)
			continue
		}
		integrations = append(integrations, integration)
	}

	log.Printf("Encontradas %d integrações relacionadas", len(integrations))
	return integrations, nil
}

// GetIMSByDate retrieves integrations being sent by date
func (r *IntegrationMarketingStructureRepositoryImpl) GetIMSByDate(date time.Time) ([]entities.IntegrationMarketingStructure, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		SELECT ID_INTEGRACAO_ESTRUTURA_MERCADOLOGICA, ID_REVENDEDOR, ID_ESTRUTURA_MERCADOLOGICA, 
			   ENVIANDO, JSON, DATA_ATUALIZACAO, TRANSACAO, DATA_INICIO_ENVIO
		FROM INTEGRACAO_ESTRUTURA_MERCADOLOGICA 
		WHERE ENVIANDO = '1' 
		  AND DATA_INICIO_ENVIO <= :1`

	rows, err := r.db.QueryContext(ctx, query, date)
	if err != nil {
		log.Printf("Erro ao consultar integrações sendo enviadas por data: %v", err)
		return nil, fmt.Errorf("erro ao consultar integrações sendo enviadas: %w", err)
	}
	defer rows.Close()

	var integrations []entities.IntegrationMarketingStructure
	for rows.Next() {
		var integration entities.IntegrationMarketingStructure
		err := rows.Scan(
			&integration.IdIntegracaoEstruturaMercadologica,
			&integration.IdRevendedor,
			&integration.IdEstruturaMercadologica,
			&integration.Enviando,
			&integration.Json,
			&integration.DataAtualizacao,
			&integration.Transacao,
			&integration.DataInicioEnvio,
		)
		if err != nil {
			log.Printf("Erro ao escanear integração: %v", err)
			continue
		}
		integrations = append(integrations, integration)
	}

	log.Printf("Encontradas %d integrações sendo enviadas para a data %v", len(integrations), date)
	return integrations, nil
}

// RemoveById removes an integration by ID
func (r *IntegrationMarketingStructureRepositoryImpl) RemoveById(id int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `DELETE FROM INTEGRACAO_ESTRUTURA_MERCADOLOGICA WHERE ID_INTEGRACAO_ESTRUTURA_MERCADOLOGICA = :1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		log.Printf("Erro ao remover integração %d: %v", id, err)
		return fmt.Errorf("erro ao remover integração: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Erro ao verificar linhas afetadas: %v", err)
		return fmt.Errorf("erro ao verificar remoção: %w", err)
	}

	if rowsAffected == 0 {
		log.Printf("Nenhuma integração foi removida para ID: %d", id)
		return fmt.Errorf("integração não encontrada para remoção: %d", id)
	}

	log.Printf("Integração %d removida com sucesso", id)
	return nil
}

// RemoverTransacaoIntegracaoEstruturaMercadologica removes transactions by cutoff date
func (r *IntegrationMarketingStructureRepositoryImpl) RemoverTransacaoIntegracaoEstruturaMercadologica(dataCorte time.Time, fazExpurgo string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if fazExpurgo == "" {
		fazExpurgo = "NAO"
	}

	query := `BEGIN sp_limparintegracaoestruturamercadologicacorte(:1, :2); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte, fazExpurgo)
	if err != nil {
		log.Printf("Erro ao executar sp_limparintegracaoestruturamercadologicacorte: %v", err)
		return fmt.Errorf("erro ao remover transações de integração estrutura mercadológica: %w", err)
	}

	log.Printf("Transações de integração estrutura mercadológica removidas com sucesso para data: %v, expurgo: %s", dataCorte, fazExpurgo)
	return nil
}
