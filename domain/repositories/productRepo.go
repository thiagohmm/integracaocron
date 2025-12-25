package repositories

import (
	"database/sql"
	"fmt"

	"github.com/thiagohmm/integracaocron/domain/entities"
)

// ProductRepository handles product database operations
type ProductRepository struct {
	db *sql.DB
}

// NewProductRepository creates a new instance of ProductRepository
func NewProductRepository(db *sql.DB) *ProductRepository {
	return &ProductRepository{
		db: db,
	}
}

// GetProductByID retrieves a product by its ID
func (r *ProductRepository) GetProductByID(id int) (*entities.Product, error) {
	query := `SELECT ID_PRODUTO, ATIVO, CONTEUDO_EMBALAGEM, DESCRICAO_CUPOM, DESCRICAO_PRODUTO, 
			  DIRETORIO_ANEXO, GIFT, ID_ESTRUTURA_MERCADOLOGICA, ID_MARCA, ID_NIVEL1_ESTR_MERC, 
			  ID_NIVEL2_ESTR_MERC, ID_NIVEL3_ESTR_MERC, ID_UNIDADE_MEDIDA, MARKUP, NOTABILIDADE, 
			  OBSERVACAO, PERIODO_SHELF_LIFE, REFERENCIA_FABRICANTE, SHELF_LIFE, TIPO_PRODUTO, 
			  PRODUCAO, PITSTOP, FORA_MIX, REGIONAL, PRODU_DATA_ULTIMA_ATUALIZACAO, CODIGO_RMS, 
			  INDUSTRIA, ID_ESTRUTURA_COMPRA
			  FROM PRODUTO WHERE ID_PRODUTO = :1`

	var product entities.Product
	err := r.db.QueryRow(query, id).Scan(
		&product.IdProduto, &product.Ativo, &product.ConteudoEmbalagem, &product.DescricaoCupom,
		&product.DescricaoProduto, &product.DiretorioAnexo, &product.Gift, &product.IdEstruturaMercadologica,
		&product.IdMarca, &product.IdNivel1EstrMerc, &product.IdNivel2EstrMerc, &product.IdNivel3EstrMerc,
		&product.IdUnidadeMedida, &product.MarkUp, &product.Notabilidade, &product.Observacao,
		&product.PeriodoShelfLife, &product.ReferenciaFabricante, &product.ShelfLife, &product.TipoProduto,
		&product.Producao, &product.PitStop, &product.ForaMix, &product.Regional,
		&product.ProduDataUltimaAtualizacao, &product.CodigoRMS, &product.Industria, &product.IdEstruturaCompra,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting product by ID: %w", err)
	}

	return &product, nil
}
