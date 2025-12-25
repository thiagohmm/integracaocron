package repositories

import (
	"database/sql"
	"fmt"

	"github.com/thiagohmm/integracaocron/domain/entities"
)

// UnitOfMeasurementRepository handles unit of measurement database operations
type UnitOfMeasurementRepository struct {
	db *sql.DB
}

// NewUnitOfMeasurementRepository creates a new instance of UnitOfMeasurementRepository
func NewUnitOfMeasurementRepository(db *sql.DB) *UnitOfMeasurementRepository {
	return &UnitOfMeasurementRepository{
		db: db,
	}
}

// GetUnitOfMeasurementByID retrieves unit of measurement by ID
func (r *UnitOfMeasurementRepository) GetUnitOfMeasurementByID(id int) (*entities.UnitOfMeasurement, error) {
	query := `SELECT ID_UNIDADE_MEDIDA, CODIGO_UNIDADE_MEDIDA, DESCRICAO_UNIDADE_MEDIDA 
			  FROM UNIDADE_MEDIDA WHERE ID_UNIDADE_MEDIDA = :1`

	var unit entities.UnitOfMeasurement
	err := r.db.QueryRow(query, id).Scan(
		&unit.IdUnidadeMedida, &unit.CodigoUnidadeMedida, &unit.DescricaoUnidadeMedida,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting unit of measurement: %w", err)
	}

	return &unit, nil
}
