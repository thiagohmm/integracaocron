package repositories

import (
	"database/sql"
	"fmt"
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
		MERGE INTO INTEGRACAOEMBALAGEMSTAGING dest
		USING (SELECT :idEmbalagemProduto AS IDEMBALAGEMPRODUTO, :idDealer AS IDREVENDEDOR FROM dual) src
		ON (dest.IDEMBALAGEMPRODUTO = src.IDEMBALAGEMPRODUTO AND dest.IDREVENDEDOR = src.IDREVENDEDOR)
		WHEN MATCHED THEN
			UPDATE SET dest.JSON = :jsonPayload, dest.DATAATUALIZACAO = SYSTIMESTAMP
		WHEN NOT MATCHED THEN
			INSERT (IDEMBALAGEMPRODUTO, IDREVENDEDOR, JSON, DATAATUALIZACAO)
			VALUES (:idEmbalagemProduto, :idDealer, :jsonPayload, SYSTIMESTAMP)
	`
	_, err := r.db.Exec(query, idEmbalagemProduto, idDealer, jsonPayload)
	if err != nil {
		return fmt.Errorf("error adding or updating packaging integration staging: %w", err)
	}
	return nil
}
