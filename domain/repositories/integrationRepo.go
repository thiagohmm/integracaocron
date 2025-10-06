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

	var query string
	if len(expurgo) > 0 && expurgo[0] == "SIM" {
		query = `DELETE FROM INTEGR_COMBO WHERE DATA_INTEGRACAO < :1`
	} else {
		query = `UPDATE INTEGR_COMBO SET STATUS_PROCESSAMENTO = 'REMOVIDO' WHERE DATA_INTEGRACAO < :1`
	}

	_, err := r.db.ExecContext(ctx, query, dataCorte)
	if err != nil {
		log.Printf("Erro ao remover integração combo: %v", err)
		return fmt.Errorf("erro ao remover integração combo: %w", err)
	}

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

// Data movement methods (stubs - implement based on your specific needs)
func (r *IntegrationRepositoryImpl) MoveIntegrationMarketingStructure(dataCorte time.Time) error {
	// TODO: Implement based on your business logic
	log.Printf("MoveIntegrationMarketingStructure called with dataCorte: %v", dataCorte)
	return nil
}

func (r *IntegrationRepositoryImpl) MoveIntegrationProductStaging(dataCorte time.Time) error {
	// TODO: Implement based on your business logic
	log.Printf("MoveIntegrationProductStaging called with dataCorte: %v", dataCorte)
	return nil
}

func (r *IntegrationRepositoryImpl) MoveIntegrationPackagingStaging(dataCorte time.Time) error {
	// TODO: Implement based on your business logic
	log.Printf("MoveIntegrationPackagingStaging called with dataCorte: %v", dataCorte)
	return nil
}

func (r *IntegrationRepositoryImpl) MoveIntegrationComboStaging(dataCorte time.Time) error {
	// TODO: Implement based on your business logic
	log.Printf("MoveIntegrationComboStaging called with dataCorte: %v", dataCorte)
	return nil
}

func (r *IntegrationRepositoryImpl) MoveIntegrationPromotionStaging(dataCorte time.Time) error {
	// TODO: Implement based on your business logic
	log.Printf("MoveIntegrationPromotionStaging called with dataCorte: %v", dataCorte)
	return nil
}

// Expiry methods
func (r *IntegrationRepositoryImpl) GetIntegrationUpdateComboByDate(dataCorte time.Time) ([]entities.IntegrationCombo, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `SELECT ID_INTEGRACAO_COMBO, DATA_INTEGRACAO FROM INTEGR_COMBO WHERE DATA_INTEGRACAO < :1`

	rows, err := r.db.QueryContext(ctx, query, dataCorte)
	if err != nil {
		log.Printf("Erro ao consultar combos para expurgo: %v", err)
		return nil, fmt.Errorf("erro ao consultar combos: %w", err)
	}
	defer rows.Close()

	var combos []entities.IntegrationCombo
	for rows.Next() {
		var combo entities.IntegrationCombo
		err := rows.Scan(&combo.IdIntegracaoCombo, &combo.DataIntegracao)
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

	query := `UPDATE SOLICITACOES SET STATUS = 'EXPIRADO' WHERE SLA_EXPIRACAO < SYSDATE AND STATUS = 'ATIVO'`

	_, err := r.db.ExecContext(ctx, query)
	if err != nil {
		log.Printf("Erro ao atualizar solicitações expiradas: %v", err)
		return fmt.Errorf("erro ao atualizar solicitações expiradas: %w", err)
	}

	return nil
}
