package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	"github.com/thiagohmm/integracaocron/domain/entities"
)

// ParameterRepositoryImpl implements the ParameterRepository interface
type ParameterRepositoryImpl struct {
	db *sql.DB
}

// NewParameterRepository creates a new instance of ParameterRepository
func NewParameterRepository(db *sql.DB) entities.ParameterRepository {
	return &ParameterRepositoryImpl{
		db: db,
	}
}

// ListByCodeParameter retrieves a parameter by its code
func (r *ParameterRepositoryImpl) ListByCodeParameter(codigo string) (*entities.IParameter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `SELECT CODIGO, VALOR, AMBIENTE FROM PARAMETROS WHERE CODIGO = :1`

	var param entities.IParameter
	err := r.db.QueryRowContext(ctx, query, codigo).Scan(
		&param.Codigo,
		&param.Valor,
		&param.Ambiente,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Parâmetro não encontrado: %s", codigo)
			return nil, nil // Return nil instead of error for not found
		}
		log.Printf("Erro ao consultar parâmetro %s: %v", codigo, err)
		return nil, fmt.Errorf("erro ao consultar parâmetro: %w", err)
	}

	log.Printf("Parâmetro encontrado: %+v", param)
	return &param, nil
}

// Update updates a parameter
func (r *ParameterRepositoryImpl) Update(param *entities.IParameter) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `UPDATE PARAMETROS SET VALOR = :1 WHERE CODIGO = :2 AND AMBIENTE = :3`

	result, err := r.db.ExecContext(ctx, query, param.Valor, param.Codigo, param.Ambiente)
	if err != nil {
		log.Printf("Erro ao atualizar parâmetro %s: %v", param.Codigo, err)
		return fmt.Errorf("erro ao atualizar parâmetro: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Erro ao verificar linhas afetadas: %v", err)
		return fmt.Errorf("erro ao verificar atualização: %w", err)
	}

	if rowsAffected == 0 {
		log.Printf("Nenhum parâmetro foi atualizado para código: %s", param.Codigo)
		return fmt.Errorf("parâmetro não encontrado para atualização: %s", param.Codigo)
	}

	log.Printf("Parâmetro %s atualizado com sucesso", param.Codigo)
	return nil
}
