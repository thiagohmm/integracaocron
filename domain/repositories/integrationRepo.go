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

	// Set default value
	fazExpurgo := "NAO"
	if len(expurgo) > 0 {
		fazExpurgo = expurgo[0]
	}

	query := `BEGIN sp_LimparIntegracaoEmbalagemCorte(:1, :2); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte, fazExpurgo)
	if err != nil {
		log.Printf("Erro ao limpar integração embalagem: %v", err)
		return fmt.Errorf("erro ao limpar integração embalagem: %w", err)
	}

	return nil
}

func (r *IntegrationRepositoryImpl) RemoverTransacaoIntegracaoEstruturaMercadologica(dataCorte time.Time, expurgo ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Set default value
	fazExpurgo := "NAO"
	if len(expurgo) > 0 {
		fazExpurgo = expurgo[0]
	}

	query := `BEGIN sp_limparintegracaoestruturamercadologicacorte(:1, :2); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte, fazExpurgo)
	if err != nil {
		log.Printf("Erro ao remover transação estrutura mercadológica: %v", err)
		return fmt.Errorf("erro ao remover transação estrutura mercadológica: %w", err)
	}

	return nil
}

func (r *IntegrationRepositoryImpl) RemoverTransacaoIntegracaoProduto(dataCorte time.Time, expurgo ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Set default value
	fazExpurgo := "NAO"
	if len(expurgo) > 0 {
		fazExpurgo = expurgo[0]
	}

	query := `BEGIN sp_limparintegracaoprodutocorte(:1, :2); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte, fazExpurgo)
	if err != nil {
		log.Printf("Erro ao remover transação produto: %v", err)
		return fmt.Errorf("erro ao remover transação produto: %w", err)
	}

	return nil
}

func (r *IntegrationRepositoryImpl) RemoverTransacaoIntegracaoPromocao(dataCorte time.Time, expurgo ...string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Set default value
	fazExpurgo := "NAO"
	if len(expurgo) > 0 {
		fazExpurgo = expurgo[0]
	}

	query := `BEGIN sp_limparintegracaopromocaocorte(:1, :2); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte, fazExpurgo)
	if err != nil {
		log.Printf("Erro ao remover transação promoção: %v", err)
		return fmt.Errorf("erro ao remover transação promoção: %w", err)
	}

	return nil
}

func (r *IntegrationRepositoryImpl) CheckMarketingStructure() (bool, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
