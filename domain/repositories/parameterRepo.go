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

	query := `SELECT ID_PARAMETRO, AMBIENTE, CODIGO, VALOR, DESCRICAO FROM PARAMETROS WHERE CODIGO = :1`

	var param entities.IParameter
	err := r.db.QueryRowContext(ctx, query, codigo).Scan(
		&param.IdParametro,
		&param.Ambiente,
		&param.Codigo,
		&param.Valor,
		&param.Descricao,
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

	query := `UPDATE PARAMETROS SET AMBIENTE = :1, CODIGO = :2, VALOR = :3, DESCRICAO = :4 WHERE ID_PARAMETRO = :5`

	result, err := r.db.ExecContext(ctx, query, param.Ambiente, param.Codigo, param.Valor, param.Descricao, param.IdParametro)
	if err != nil {
		log.Printf("Erro ao atualizar parâmetro %d: %v", param.IdParametro, err)
		return fmt.Errorf("erro ao atualizar parâmetro: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Erro ao verificar linhas afetadas: %v", err)
		return fmt.Errorf("erro ao verificar atualização: %w", err)
	}

	if rowsAffected == 0 {
		log.Printf("Nenhum parâmetro foi atualizado para ID: %d", param.IdParametro)
		return fmt.Errorf("parâmetro não encontrado para atualização: %d", param.IdParametro)
	}

	log.Printf("Parâmetro %d atualizado com sucesso", param.IdParametro)
	return nil
}

// Delete deletes a parameter by ID
func (r *ParameterRepositoryImpl) Delete(idParametro int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `DELETE FROM PARAMETROS WHERE ID_PARAMETRO = :1`

	result, err := r.db.ExecContext(ctx, query, idParametro)
	if err != nil {
		log.Printf("Erro ao deletar parâmetro %d: %v", idParametro, err)
		return fmt.Errorf("erro ao deletar parâmetro: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Erro ao verificar linhas afetadas: %v", err)
		return fmt.Errorf("erro ao verificar exclusão: %w", err)
	}

	if rowsAffected == 0 {
		log.Printf("Nenhum parâmetro foi deletado para ID: %d", idParametro)
		return fmt.Errorf("parâmetro não encontrado para exclusão: %d", idParametro)
	}

	log.Printf("Parâmetro %d deletado com sucesso", idParametro)
	return nil
}

// ListById retrieves a parameter by its ID
func (r *ParameterRepositoryImpl) ListById(idParametro int) (*entities.IParameter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `SELECT ID_PARAMETRO, AMBIENTE, CODIGO, VALOR, DESCRICAO FROM PARAMETROS WHERE ID_PARAMETRO = :1`

	var param entities.IParameter
	err := r.db.QueryRowContext(ctx, query, idParametro).Scan(
		&param.IdParametro,
		&param.Ambiente,
		&param.Codigo,
		&param.Valor,
		&param.Descricao,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Parâmetro não encontrado com ID: %d", idParametro)
			return nil, nil // Return nil instead of error for not found
		}
		log.Printf("Erro ao consultar parâmetro %d: %v", idParametro, err)
		return nil, fmt.Errorf("erro ao consultar parâmetro: %w", err)
	}

	log.Printf("Parâmetro encontrado: %+v", param)
	return &param, nil
}

// ListGridPerFilter retrieves parameters based on filter criteria
func (r *ParameterRepositoryImpl) ListGridPerFilter(filter *entities.IFilterParameter) ([]entities.IParameter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	baseQuery := `SELECT ID_PARAMETRO, AMBIENTE, CODIGO, VALOR, DESCRICAO FROM PARAMETROS WHERE 1=1`
	var args []interface{}
	argCount := 0

	// Build dynamic WHERE clause based on filter
	if filter.Codigo != "" {
		argCount++
		baseQuery += fmt.Sprintf(" AND UPPER(CODIGO) LIKE UPPER(:%d)", argCount)
		args = append(args, "%"+filter.Codigo+"%")
	}

	if filter.Ambiente != "" {
		argCount++
		baseQuery += fmt.Sprintf(" AND UPPER(AMBIENTE) LIKE UPPER(:%d)", argCount)
		args = append(args, "%"+filter.Ambiente+"%")
	}

	baseQuery += " ORDER BY CODIGO"

	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		log.Printf("Erro ao consultar parâmetros com filtro: %v", err)
		return nil, fmt.Errorf("erro ao consultar parâmetros: %w", err)
	}
	defer rows.Close()

	var parameters []entities.IParameter
	for rows.Next() {
		var param entities.IParameter
		err := rows.Scan(
			&param.IdParametro,
			&param.Ambiente,
			&param.Codigo,
			&param.Valor,
			&param.Descricao,
		)
		if err != nil {
			log.Printf("Erro ao escanear parâmetro: %v", err)
			continue
		}
		parameters = append(parameters, param)
	}

	log.Printf("Encontrados %d parâmetros com o filtro aplicado", len(parameters))
	return parameters, nil
}

// Create creates a new parameter
func (r *ParameterRepositoryImpl) Create(param *entities.IParameter) (*entities.IParameter, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `INSERT INTO PARAMETROS (AMBIENTE, CODIGO, VALOR, DESCRICAO) 
			  VALUES (:1, :2, :3, :4) 
			  RETURNING ID_PARAMETRO INTO :5`

	var newID int
	_, err := r.db.ExecContext(ctx, query, param.Ambiente, param.Codigo, param.Valor, param.Descricao, &newID)
	if err != nil {
		log.Printf("Erro ao criar parâmetro: %v", err)
		return nil, fmt.Errorf("erro ao criar parâmetro: %w", err)
	}

	param.IdParametro = newID
	log.Printf("Parâmetro criado com sucesso, ID: %d", newID)
	return param, nil
}
