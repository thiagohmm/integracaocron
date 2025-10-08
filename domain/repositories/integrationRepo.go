package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/thiagohmm/integracaocron/domain/entities"
)

// IntegrationRepositoryImpl implements the IntegrationRepository interface
type IntegrationRepositoryImpl struct {
	db *sql.DB
}

// NewIntegrationRepository creates a new instance of IntegrationRepository
func NewIntegrationRepository(db *sql.DB) entities.IntegrationRepository {
	return &IntegrationRepositoryImpl{
		db: db,
	}
}

// Transaction removal methods
func (r *IntegrationRepositoryImpl) RemoveIntegrationCombo(dataCorte time.Time, expurgo ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Set default value for expurgo if not provided
	fazExpurgo := "NAO"
	if len(expurgo) > 0 {
		fazExpurgo = expurgo[0]
	}

	query := `BEGIN sp_limparintegracaocombocorte(:1, :2); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte, fazExpurgo)
	if err != nil {
		log.Printf("Erro ao executar sp_limparintegracaocombocorte: %v", err)
		return fmt.Errorf("erro ao remover integração combo: %w", err)
	}

	log.Printf("Integração combo removida com sucesso para data corte: %v, expurgo: %s", dataCorte, fazExpurgo)
	return nil
}

func (r *IntegrationRepositoryImpl) ClearIntegrationPackagingByCutOffDate(dataCorte time.Time, expurgo ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var query string
	if len(expurgo) > 0 && expurgo[0] == "SIM" {
		query = `DELETE FROM INTEGR_EMBALAGEM WHERE DATA_INTEGRACAO < :1`
	} else {
		query = `UPDATE INTEGR_EMBALAGEM SET STATUS_PROCESSAMENTO = 'REMOVIDO' WHERE DATA_INTEGRACAO < :1`
	}

	_, err := r.db.ExecContext(ctx, query, dataCorte)
	if err != nil {
		log.Printf("Erro ao limpar integração embalagem: %v", err)
		return fmt.Errorf("erro ao limpar integração embalagem: %w", err)
	}

	return nil
}

func (r *IntegrationRepositoryImpl) RemoverTransacaoIntegracaoEstruturaMercadologica(dataCorte time.Time, expurgo ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var query string
	if len(expurgo) > 0 && expurgo[0] == "SIM" {
		query = `DELETE FROM INTEGR_ESTRUTURA_MERCADOLOGICA WHERE DATA_INTEGRACAO < :1`
	} else {
		query = `UPDATE INTEGR_ESTRUTURA_MERCADOLOGICA SET STATUS_PROCESSAMENTO = 'REMOVIDO' WHERE DATA_INTEGRACAO < :1`
	}

	_, err := r.db.ExecContext(ctx, query, dataCorte)
	if err != nil {
		log.Printf("Erro ao remover transação estrutura mercadológica: %v", err)
		return fmt.Errorf("erro ao remover transação estrutura mercadológica: %w", err)
	}

	return nil
}

func (r *IntegrationRepositoryImpl) RemoverTransacaoIntegracaoProduto(dataCorte time.Time, expurgo ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var query string
	if len(expurgo) > 0 && expurgo[0] == "SIM" {
		query = `DELETE FROM INTEGR_PRODUTO WHERE DATA_INTEGRACAO < :1`
	} else {
		query = `UPDATE INTEGR_PRODUTO SET STATUS_PROCESSAMENTO = 'REMOVIDO' WHERE DATA_INTEGRACAO < :1`
	}

	_, err := r.db.ExecContext(ctx, query, dataCorte)
	if err != nil {
		log.Printf("Erro ao remover transação produto: %v", err)
		return fmt.Errorf("erro ao remover transação produto: %w", err)
	}

	return nil
}

func (r *IntegrationRepositoryImpl) RemoverTransacaoIntegracaoPromocao(dataCorte time.Time, expurgo ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var query string
	if len(expurgo) > 0 && expurgo[0] == "SIM" {
		query = `DELETE FROM INTEGR_PROMOCAO WHERE DATA_INTEGRACAO < :1`
	} else {
		query = `UPDATE INTEGR_PROMOCAO SET STATUS_PROCESSAMENTO = 'REMOVIDO' WHERE DATA_INTEGRACAO < :1`
	}

	_, err := r.db.ExecContext(ctx, query, dataCorte)
	if err != nil {
		log.Printf("Erro ao remover transação promoção: %v", err)
		return fmt.Errorf("erro ao remover transação promoção: %w", err)
	}

	return nil
}

// Data movement methods
func (r *IntegrationRepositoryImpl) MoveIntegrationMarketingStructure(dataCorte time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

func (r *IntegrationRepositoryImpl) MoveIntegrationProductStaging(dataCorte time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `BEGIN sp_MoverStagingProduto(:1); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte)
	if err != nil {
		log.Printf("Erro ao executar sp_MoverStagingProduto: %v", err)
		return fmt.Errorf("erro ao mover produto para staging: %w", err)
	}

	log.Printf("Produto movido para staging com sucesso para data: %v", dataCorte)
	return nil
}

func (r *IntegrationRepositoryImpl) MoveIntegrationPackagingStaging(dataCorte time.Time) error {
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

func (r *IntegrationRepositoryImpl) MoveIntegrationComboStaging(dataCorte time.Time) error {
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

func (r *IntegrationRepositoryImpl) MoveIntegrationPromotionStaging(dataCorte time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `BEGIN sp_MoverStagingPromocao(:1); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte)
	if err != nil {
		log.Printf("Erro ao executar sp_MoverStagingPromocao: %v", err)
		return fmt.Errorf("erro ao mover promoção para staging: %w", err)
	}

	log.Printf("Promoção movida para staging com sucesso para data: %v", dataCorte)
	return nil
}

// Expiry methods
func (r *IntegrationRepositoryImpl) GetIntegrationUpdateComboByDate(dataCorte time.Time) ([]entities.IntegrationCombo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		SELECT ID_INTEGRACAO_COMBO, ID_REVENDEDOR, ID_COMBO_PROMOCAO, 
			   ENVIANDO, JSON, DATA_ATUALIZACAO, TRANSACAO, DATA_INICIO_ENVIO
		FROM INTEGR_COMBO WHERE DATA_ATUALIZACAO < :1`

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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `DELETE FROM INTEGR_COMBO WHERE ID_INTEGRACAO_COMBO = :1`

	_, err := r.db.ExecContext(ctx, query, idIntegracaoCombo)
	if err != nil {
		log.Printf("Erro ao deletar combo %d: %v", idIntegracaoCombo, err)
		return fmt.Errorf("erro ao deletar combo: %w", err)
	}

	return nil
}

func (r *IntegrationRepositoryImpl) UpdateExpiredSlaSolicitation() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
