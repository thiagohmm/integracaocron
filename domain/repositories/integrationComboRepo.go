package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/thiagohmm/integracaocron/domain/entities"
)

// IntegrationComboRepositoryImpl implements the IntegrationComboRepository interface
type IntegrationComboRepositoryImpl struct {
	db *sql.DB
}

// NewIntegrationComboRepository creates a new instance of IntegrationComboRepository
func NewIntegrationComboRepository(db *sql.DB) entities.IntegrationComboRepository {
	return &IntegrationComboRepositoryImpl{
		db: db,
	}
}

// GetIntegrationUpdateComboByDate retrieves combo integrations by update date
func (r *IntegrationComboRepositoryImpl) GetIntegrationUpdateComboByDate(date time.Time) ([]entities.IntegrationCombo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		SELECT ID_INTEGRACAO_COMBO, ID_REVENDEDOR, ID_COMBO_PROMOCAO, 
			   ENVIANDO, JSON, DATA_ATUALIZACAO, TRANSACAO, DATA_INICIO_ENVIO
		FROM INTEGRACAO_COMBO 
		WHERE DATA_ATUALIZACAO <= :1`

	rows, err := r.db.QueryContext(ctx, query, date)
	if err != nil {
		log.Printf("Erro ao consultar integrações de combo por data de atualização: %v", err)
		return nil, fmt.Errorf("erro ao consultar integrações de combo: %w", err)
	}
	defer rows.Close()

	var integrations []entities.IntegrationCombo
	for rows.Next() {
		var integration entities.IntegrationCombo
		err := rows.Scan(
			&integration.IdIntegracaoCombo,
			&integration.IdRevendedor,
			&integration.IdComboPromocao,
			&integration.Enviando,
			&integration.Json,
			&integration.DataAtualizacao,
			&integration.Transacao,
			&integration.DataInicioEnvio,
		)
		if err != nil {
			log.Printf("Erro ao escanear integração de combo: %v", err)
			continue
		}
		integrations = append(integrations, integration)
	}

	log.Printf("Encontradas %d integrações de combo para a data %v", len(integrations), date)
	return integrations, nil
}

// GetIntegrationComboByDate retrieves combo integrations being sent by date
func (r *IntegrationComboRepositoryImpl) GetIntegrationComboByDate(date time.Time) ([]entities.IntegrationCombo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		SELECT ID_INTEGRACAO_COMBO, ID_REVENDEDOR, ID_COMBO_PROMOCAO, 
			   ENVIANDO, JSON, DATA_ATUALIZACAO, TRANSACAO, DATA_INICIO_ENVIO
		FROM INTEGRACAO_COMBO 
		WHERE ENVIANDO = '1' 
		  AND DATA_INICIO_ENVIO <= :1`

	rows, err := r.db.QueryContext(ctx, query, date)
	if err != nil {
		log.Printf("Erro ao consultar integrações de combo sendo enviadas por data: %v", err)
		return nil, fmt.Errorf("erro ao consultar integrações de combo: %w", err)
	}
	defer rows.Close()

	var integrations []entities.IntegrationCombo
	for rows.Next() {
		var integration entities.IntegrationCombo
		err := rows.Scan(
			&integration.IdIntegracaoCombo,
			&integration.IdRevendedor,
			&integration.IdComboPromocao,
			&integration.Enviando,
			&integration.Json,
			&integration.DataAtualizacao,
			&integration.Transacao,
			&integration.DataInicioEnvio,
		)
		if err != nil {
			log.Printf("Erro ao escanear integração de combo: %v", err)
			continue
		}
		integrations = append(integrations, integration)
	}

	log.Printf("Encontradas %d integrações de combo sendo enviadas para a data %v", len(integrations), date)
	return integrations, nil
}

// RemoveIntegrationCombo removes combo integrations by cutoff date
func (r *IntegrationComboRepositoryImpl) RemoveIntegrationCombo(date time.Time, doPurge string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if doPurge == "" {
		doPurge = "NAO"
	}

	query := `BEGIN sp_limparintegracaocombocorte(:1, :2); END;`

	_, err := r.db.ExecContext(ctx, query, date, doPurge)
	if err != nil {
		log.Printf("Erro ao executar sp_limparintegracaocombocorte: %v", err)
		return fmt.Errorf("erro ao remover integrações de combo: %w", err)
	}

	log.Printf("Integrações de combo removidas com sucesso para data: %v, expurgo: %s", date, doPurge)
	return nil
}

// MoveIntegrationComboStaging moves combo integrations to staging
func (r *IntegrationComboRepositoryImpl) MoveIntegrationComboStaging(dataCorte time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
