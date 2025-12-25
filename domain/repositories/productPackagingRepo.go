package repositories

import (
	"database/sql"
	"fmt"

	"github.com/thiagohmm/integracaocron/domain/entities"
)

// ProductPackagingRepository handles product packaging database operations
type ProductPackagingRepository struct {
	db *sql.DB
}

// NewProductPackagingRepository creates a new instance of ProductPackagingRepository
func NewProductPackagingRepository(db *sql.DB) *ProductPackagingRepository {
	return &ProductPackagingRepository{
		db: db,
	}
}

// ListByIdProductPackaging retrieves all packaging for a given product ID
func (r *ProductPackagingRepository) ListByIdProductPackaging(idProduto int) ([]entities.ProductPackaging, error) {
	query := `SELECT ID_PRODUTO, CODIGO_BARRAS, PRINCIPAL, QUANTIDADE_EMBALAGEM, ID_UNIDADE_MEDIDA, TIPO_CODIGO_BARRAS 
			  FROM EMBALAGEM_PRODUTO WHERE ID_PRODUTO = :1`

	rows, err := r.db.Query(query, idProduto)
	if err != nil {
		return nil, fmt.Errorf("error querying product packaging: %w", err)
	}
	defer rows.Close()

	var results []entities.ProductPackaging
	for rows.Next() {
		var pkg entities.ProductPackaging
		err := rows.Scan(
			&pkg.IdProduto, &pkg.CodigoBarras, &pkg.Principal,
			&pkg.QuantidadeEmbalagem, &pkg.IdUnidadeMedida, &pkg.TipoCodigoBarras,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning product packaging row: %w", err)
		}
		results = append(results, pkg)
	}

	return results, nil
}
