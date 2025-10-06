package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/thiagohmm/integracaocron/domain/entities"
)

type PromotionRepositoryImpl struct {
	db *sql.DB
}

func NewPromotionRepository(db *sql.DB) entities.PromotionRepository {
	return &PromotionRepositoryImpl{
		db: db,
	}
}

// Dopkg_promotion executes the Oracle stored procedure pkg_integra_promocao.prc_integra_hermes
func (r *PromotionRepositoryImpl) Dopkg_promotion(pIprId int) (*entities.PromotionResult, error) {
	// Create context with timeout for the database operation
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Prepare the PL/SQL block to call the stored procedure
	query := "BEGIN pkg_integra_promocao.prc_integra_hermes(:parametro1); END;"

	// Execute the stored procedure
	_, err := r.db.ExecContext(ctx, query, pIprId)
	if err != nil {
		log.Printf("Erro ao executar pkg_integra_promocao.prc_integra_hermes: %v", err)
		return &entities.PromotionResult{
			Success: false,
			Message: fmt.Sprintf("Erro ao executar procedimento: %v", err),
		}, nil
	}

	// Return success result
	return &entities.PromotionResult{
		Success: true,
		Message: "Processamento realizado com sucesso.",
	}, nil
}

// GetIntegrRMSPromocaoIN retrieves promotion data for integration
func (r *PromotionRepositoryImpl) GetIntegrRMSPromocaoIN() ([]entities.Promotion, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `SELECT IPMD_ID, JSON_DATA, DATARECEBIMENTO 
			  FROM INTEGR_RMS_PROMOCAO_IN 
			  ORDER BY DATARECEBIMENTO`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Erro ao consultar promoções para integração: %v", err)
		return nil, fmt.Errorf("erro ao consultar promoções: %w", err)
	}
	defer rows.Close()

	var promotions []entities.Promotion
	for rows.Next() {
		var promo entities.Promotion
		var jsonData sql.NullString
		var dataRecebimento sql.NullString

		err := rows.Scan(&promo.IPMD_ID, &jsonData, &dataRecebimento)
		if err != nil {
			log.Printf("Erro ao escanear linha de promoção: %v", err)
			continue
		}

		promo.Json = jsonData.String
		promo.DATARECEBIMENTO = dataRecebimento.String

		promotions = append(promotions, promo)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Erro durante iteração das linhas: %v", err)
		return nil, fmt.Errorf("erro durante iteração: %w", err)
	}

	log.Printf("Encontradas %d promoções para integração", len(promotions))
	return promotions, nil
}

// DeletePorObjeto deletes a promotion record by IPMD_ID
func (r *PromotionRepositoryImpl) DeletePorObjeto(ipmID int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := "DELETE FROM INTEGR_RMS_PROMOCAO_IN WHERE IPMD_ID = :1"

	result, err := r.db.ExecContext(ctx, query, ipmID)
	if err != nil {
		log.Printf("Erro ao deletar promoção %d: %v", ipmID, err)
		return fmt.Errorf("erro ao deletar promoção: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Erro ao verificar linhas afetadas: %v", err)
		return fmt.Errorf("erro ao verificar deleção: %w", err)
	}

	if rowsAffected == 0 {
		log.Printf("Nenhuma promoção encontrada para deletar com ID: %d", ipmID)
		return fmt.Errorf("promoção não encontrada para deleção: %d", ipmID)
	}

	log.Printf("Promoção %d deletada com sucesso", ipmID)
	return nil
}
