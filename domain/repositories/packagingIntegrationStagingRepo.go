package repositories

import (
	"database/sql"
	"fmt"

	"github.com/thiagohmm/integracaocron/domain/entities"
)

// PackagingIntegrationStagingRepository handles packaging integration staging database operations
type PackagingIntegrationStagingRepository struct {
	db *sql.DB
}

// NewPackagingIntegrationStagingRepository creates a new instance of PackagingIntegrationStagingRepository
func NewPackagingIntegrationStagingRepository(db *sql.DB) *PackagingIntegrationStagingRepository {
	return &PackagingIntegrationStagingRepository{
		db: db,
	}
}

// AddOrUpdate adds or updates a packaging integration staging record
func (r *PackagingIntegrationStagingRepository) AddOrUpdate(idEmbalagemProduto, idDealer int, jsonPayload string) error {
	query := `
		MERGE INTO INTEGRACAO_EMBALAGEM_STAGING dest
		USING (SELECT :idEmbalagemProduto AS ID_EMBALAGEM_PRODUTO, :idDealer AS ID_REVENDEDOR FROM dual) src
		ON (dest.ID_EMBALAGEM_PRODUTO = src.ID_EMBALAGEM_PRODUTO AND dest.ID_REVENDEDOR = src.ID_REVENDEDOR)
		WHEN MATCHED THEN
			UPDATE SET dest.JSON = :jsonPayload, dest.DATA_ATUALIZACAO = SYSDATE
		WHEN NOT MATCHED THEN
			INSERT (ID_EMBALAGEM_PRODUTO, ID_REVENDEDOR, JSON, DATA_ATUALIZACAO)
			VALUES (:idEmbalagemProduto, :idDealer, :jsonPayload, SYSDATE)
	`
	_, err := r.db.Exec(query, idEmbalagemProduto, idDealer, jsonPayload)
	if err != nil {
		return fmt.Errorf("error adding or updating packaging integration staging: %w", err)
	}
	return nil
}
