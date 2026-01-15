package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/thiagohmm/integracaocron/domain/entities"
)

// ProductIntegrationRepository handles product integration database operations
type ProductIntegrationRepository struct {
	db *sql.DB
}

// NewProductIntegrationRepository creates a new instance of ProductIntegrationRepository
func NewProductIntegrationRepository(db *sql.DB) *ProductIntegrationRepository {
	return &ProductIntegrationRepository{
		db: db,
	}
}

// GetIntegrRmsProductsIn retrieves all pending RMS product integrations
func (r *ProductIntegrationRepository) GetIntegrRmsProductsIn() ([]entities.IntegrRmsProductIn, error) {
	query := `SELECT IPR_ID, JSON, DATARECEBIMENTO FROM INTEGRRMSPRODUTOIN ORDER BY DATARECEBIMENTO ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying integrrmsprodutoin: %w", err)
	}
	defer rows.Close()

	var results []entities.IntegrRmsProductIn
	for rows.Next() {
		var item entities.IntegrRmsProductIn
		err := rows.Scan(&item.IprID, &item.JSON, &item.DataRecebimento)
		if err != nil {
			return nil, fmt.Errorf("error scanning row: %w", err)
		}
		results = append(results, item)
	}

	return results, nil
}

// RemoveProductService removes a processed product integration record
func (r *ProductIntegrationRepository) RemoveProductService(rms entities.IntegrRmsProductIn) error {
	// Iniciar transação
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}
	defer tx.Rollback() // Rollback automático se não der commit

	// Executar DELETE
	query := `DELETE FROM INTEGRRMSPRODUTOIN WHERE IPR_ID = :1`
	result, err := tx.Exec(query, rms.IprID)
	if err != nil {
		return fmt.Errorf("error removing product service: %w", err)
	}

	// Verificar se deletou algo
	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		log.Printf("⚠️  No rows deleted for IPR_ID: %v", rms.IprID)
	}

	// Commit da transação
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing delete: %w", err)
	}

	return nil
}

// GetMarketingStructureLevel2 retrieves marketing structure level 2 information
func (r *ProductIntegrationRepository) GetMarketingStructureLevel2(idLevel2 int) (*entities.MarketingStructure, error) {
	query := `SELECT ID_ESTRUTURA_MERCADOLOGICA, ID_NIVEL_PAI, ID_DEPARTAMENTO, ID_SECAO, DESCRICAO_ESTRUTURA 
			  FROM ESTRUTURA_MERCADOLOGICA WHERE ID_ESTRUTURA_MERCADOLOGICA = :1`

	var ms entities.MarketingStructure
	err := r.db.QueryRow(query, idLevel2).Scan(
		&ms.IdEstruturaMercadologica,
		&ms.IdNivelPai,
		&ms.IdDepartamento,
		&ms.IdSecao,
		&ms.DescricaoEstrutura,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting marketing structure level 2: %w", err)
	}

	return &ms, nil
}

// GetMarketingStructureLevel4 retrieves marketing structure level 4 information
func (r *ProductIntegrationRepository) GetMarketingStructureLevel4(idLevel4 int) ([]entities.MarketingStructure, error) {
	query := `SELECT ID_ESTRUTURA_MERCADOLOGICA, ID_NIVEL_PAI, ID_DEPARTAMENTO, ID_SECAO, DESCRICAO_ESTRUTURA 
			  FROM ESTRUTURA_MERCADOLOGICA WHERE ID_ESTRUTURA_MERCADOLOGICA = :1`

	rows, err := r.db.Query(query, idLevel4)
	if err != nil {
		return nil, fmt.Errorf("error querying marketing structure level 4: %w", err)
	}
	defer rows.Close()

	var results []entities.MarketingStructure
	for rows.Next() {
		var ms entities.MarketingStructure
		err := rows.Scan(&ms.IdEstruturaMercadologica, &ms.IdNivelPai, &ms.IdDepartamento, &ms.IdSecao, &ms.DescricaoEstrutura)
		if err != nil {
			return nil, fmt.Errorf("error scanning marketing structure row: %w", err)
		}
		results = append(results, ms)
	}

	return results, nil
}

// GetBrandByIndustryName retrieves brands by industry and name
func (r *ProductIntegrationRepository) GetBrandByIndustryName(brandName, industryName string) ([]entities.Brand, error) {
	query := `SELECT m.ID_MARCA, m.NOME_MARCA, m.ID_INDUSTRIA, m.STATUS_MARCA, i.NOME_INDUSTRIA 
			  FROM MARCA m 
			  JOIN INDUSTRIA i ON m.ID_INDUSTRIA = i.ID_INDUSTRIA 
			  WHERE UPPER(m.NOME_MARCA) = UPPER(:1) AND UPPER(i.NOME_INDUSTRIA) = UPPER(:2)`

	rows, err := r.db.Query(query, brandName, industryName)
	if err != nil {
		return nil, fmt.Errorf("error querying brands: %w", err)
	}
	defer rows.Close()

	var results []entities.Brand
	for rows.Next() {
		var brand entities.Brand
		err := rows.Scan(&brand.IdMarca, &brand.NomeMarca, &brand.IdIndustria, &brand.StatusMarca, &brand.NomeIndustria)
		if err != nil {
			return nil, fmt.Errorf("error scanning brand row: %w", err)
		}
		results = append(results, brand)
	}

	return results, nil
}

// GetIndustryByNameAndStatus retrieves industry by name and status
func (r *ProductIntegrationRepository) GetIndustryByNameAndStatus(nomeIndustria string, statusIndustria int) (*entities.Industry, error) {
	query := `SELECT ID_INDUSTRIA, NOME_INDUSTRIA, STATUS_INDUSTRIA 
			  FROM INDUSTRIA 
			  WHERE UPPER(NOME_INDUSTRIA) = UPPER(:1) AND STATUS_INDUSTRIA = :2`

	var industry entities.Industry
	err := r.db.QueryRow(query, nomeIndustria, statusIndustria).Scan(
		&industry.IdIndustria,
		&industry.NomeIndustria,
		&industry.StatusIndustria,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting industry: %w", err)
	}

	return &industry, nil
}

// SaveIndustry saves a new industry
func (r *ProductIntegrationRepository) SaveIndustry(industry entities.Industry) (*entities.Industry, error) {
	query := `INSERT INTO INDUSTRIA (NOME_INDUSTRIA, STATUS_INDUSTRIA) 
			  VALUES (:1, :2) RETURNING ID_INDUSTRIA INTO :3`

	var newID int
	_, err := r.db.Exec(query, industry.NomeIndustria, industry.StatusIndustria, &newID)
	if err != nil {
		return nil, fmt.Errorf("error saving industry: %w", err)
	}

	industry.IdIndustria = &newID
	return &industry, nil
}

// SaveBrand saves a new brand
func (r *ProductIntegrationRepository) SaveBrand(brand entities.Brand) (*entities.Brand, error) {
	query := `INSERT INTO MARCA (NOME_MARCA, ID_INDUSTRIA, STATUS_MARCA) 
			  VALUES (:1, :2, :3) RETURNING ID_MARCA INTO :4`

	var newID int
	_, err := r.db.Exec(query, brand.NomeMarca, brand.IdIndustria, brand.StatusMarca, &newID)
	if err != nil {
		return nil, fmt.Errorf("error saving brand: %w", err)
	}

	brand.IdMarca = &newID
	return &brand, nil
}

// GetProductByCodeRMS retrieves product by RMS code
func (r *ProductIntegrationRepository) GetProductByCodeRMS(codeRms int) (*entities.Product, error) {
	query := `SELECT ID_PRODUTO, ATIVO, CONTEUDO_EMBALAGEM, DESCRICAO_CUPOM, DESCRICAO_PRODUTO, 
			  DIRETORIO_ANEXO, GIFT, ID_ESTRUTURA_MERCADOLOGICA, ID_MARCA, ID_NIVEL1_ESTR_MERC, 
			  ID_NIVEL2_ESTR_MERC, ID_NIVEL3_ESTR_MERC, ID_UNIDADE_MEDIDA, MARKUP, NOTABILIDADE, 
			  OBSERVACAO, PERIODO_SHELF_LIFE, REFERENCIA_FABRICANTE, SHELF_LIFE, TIPO_PRODUTO, 
			  PRODUCAO, PITSTOP, FORA_MIX, REGIONAL, PRODU_DATA_ULTIMA_ATUALIZACAO, CODIGO_RMS, 
			  INDUSTRIA, ID_ESTRUTURA_COMPRA
			  FROM PRODUTO WHERE CODIGO_RMS = :1`

	var product entities.Product
	err := r.db.QueryRow(query, codeRms).Scan(
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
		return nil, fmt.Errorf("error getting product by RMS code: %w", err)
	}

	return &product, nil
}

// GetProductPackagingByBarCode retrieves product packaging by barcode
func (r *ProductIntegrationRepository) GetProductPackagingByBarCode(barCode string) (*entities.ProductPackaging, error) {
	query := `SELECT ID_PRODUTO, CODIGO_BARRAS, PRINCIPAL, QUANTIDADE_EMBALAGEM, ID_UNIDADE_MEDIDA, TIPO_CODIGO_BARRAS 
			  FROM EMBALAGEM_PRODUTO WHERE CODIGO_BARRAS = :1`

	var pkg entities.ProductPackaging
	err := r.db.QueryRow(query, barCode).Scan(
		&pkg.IdProduto, &pkg.CodigoBarras, &pkg.Principal,
		&pkg.QuantidadeEmbalagem, &pkg.IdUnidadeMedida, &pkg.TipoCodigoBarras,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("error getting product packaging by barcode: %w", err)
	}

	return &pkg, nil
}

// GetDepartmentNameByID retrieves department name by ID
func (r *ProductIntegrationRepository) GetDepartmentNameByID(id *int) (string, error) {
	if id == nil {
		return "Não encontrado", nil
	}

	query := `SELECT NOME_DEPARTAMENTO FROM DEPARTAMENTO WHERE ID_DEPARTAMENTO = :1`

	var name string
	err := r.db.QueryRow(query, *id).Scan(&name)
	if err != nil {
		if err == sql.ErrNoRows {
			return "Não encontrado", nil
		}
		return "", fmt.Errorf("error getting department name: %w", err)
	}

	return name, nil
}

// GetSectionNameByID retrieves section name by ID
func (r *ProductIntegrationRepository) GetSectionNameByID(id *int) (*entities.Section, error) {
	if id == nil {
		return &entities.Section{NomeSecao: "Não encontrado"}, nil
	}

	query := `SELECT ID_SECAO, NOME_SECAO FROM SECAO WHERE ID_SECAO = :1`

	var section entities.Section
	err := r.db.QueryRow(query, *id).Scan(&section.IdSecao, &section.NomeSecao)
	if err != nil {
		if err == sql.ErrNoRows {
			return &entities.Section{NomeSecao: "Não encontrado"}, nil
		}
		return nil, fmt.Errorf("error getting section name: %w", err)
	}

	return &section, nil
}

// GetBrandDescByID retrieves brand description by ID
func (r *ProductIntegrationRepository) GetBrandDescByID(id *int) ([]entities.Brand, error) {
	if id == nil {
		return []entities.Brand{}, nil
	}

	query := `SELECT ID_MARCA, NOME_MARCA, ID_INDUSTRIA, STATUS_MARCA FROM MARCA WHERE ID_MARCA = :1`

	rows, err := r.db.Query(query, *id)
	if err != nil {
		return nil, fmt.Errorf("error querying brand description: %w", err)
	}
	defer rows.Close()

	var results []entities.Brand
	for rows.Next() {
		var brand entities.Brand
		err := rows.Scan(&brand.IdMarca, &brand.NomeMarca, &brand.IdIndustria, &brand.StatusMarca)
		if err != nil {
			return nil, fmt.Errorf("error scanning brand row: %w", err)
		}
		results = append(results, brand)
	}

	return results, nil
}

// DoPackageProductIntegration executes Oracle stored procedure for product integration
// A procedure Oracle espera que o registro exista em INTEGRRMSPRODUTOIN com JSON válido
func (r *ProductIntegrationRepository) DoPackageProductIntegration(iprID int) (*entities.LogValidate, error) {
	log.Printf("🔍 Validating IPR_ID %d before calling Oracle procedure", iprID)

	// 🛡️ VALIDAÇÃO CRÍTICA: Verificar se o JSON existe e não está vazio
	// A procedure Oracle pkg_integra_produto.prc_integra_hermes espera encontrar
	// o registro em INTEGRRMSPRODUTOIN e lança ORA-20000 se JSON estiver vazio
	var jsonData sql.NullString
	checkQuery := `SELECT JSON FROM INTEGRRMSPRODUTOIN WHERE IPR_ID = :1`
	err := r.db.QueryRow(checkQuery, iprID).Scan(&jsonData)

	if err == sql.ErrNoRows {
		// Registro não encontrado - pode ser um product_id direto, não IPR_ID
		log.Printf("⚠️ IPR_ID %d not found in INTEGRRMSPRODUTOIN, proceeding anyway (may be product_id)", iprID)
	} else if err != nil {
		log.Printf("❌ ERROR checking JSON for IPR_ID %d: %v", iprID, err)
		return &entities.LogValidate{
			Success: false,
			Message: fmt.Sprintf("Erro ao verificar JSON: %v", err),
		}, nil
	} else {
		// Registro encontrado - validar JSON
		if !jsonData.Valid || strings.TrimSpace(jsonData.String) == "" {
			msg := fmt.Sprintf("JSON vazio para IPR_ID %d - não é possível processar", iprID)
			log.Printf("❌ VALIDATION FAILED: %s", msg)

			return &entities.LogValidate{
				Success: false,
				Message: msg,
			}, nil
		}
		log.Printf("✅ JSON validation passed for IPR_ID %d (length: %d bytes)", iprID, len(jsonData.String))
	}

	// Chamar procedure Oracle
	query := `BEGIN pkg_integra_produto.prc_integra_hermes(:1); END;`
	log.Printf("📞 Executing Oracle procedure: pkg_integra_produto.prc_integra_hermes with IPR_ID: %d", iprID)

	result, err := r.db.Exec(query, iprID)
	if err != nil {
		log.Printf("❌ ERROR executing pkg_integra_produto.prc_integra_hermes for IPR_ID %d: %v", iprID, err)
		return &entities.LogValidate{
			Success: false,
			Message: err.Error(),
		}, nil
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("✅ Oracle procedure executed successfully for IPR_ID %d (rows affected: %d)", iprID, rowsAffected)

	return &entities.LogValidate{
		Success: true,
		Message: "Processamento realizado com sucesso.",
	}, nil
}

// SaveLogIntegration saves integration log
func (r *ProductIntegrationRepository) SaveLogIntegration(log entities.LogIntegrRMS) error {
	query := `INSERT INTO LOGS_INTEGR_RMS (TRANSACAO, TABELA, DATARECEBIMENTO, DATAPROCESSAMENTO, 
			  STATUSPROCESSAMENTO, JSON, DESCRICAOERRO) 
			  VALUES (:1, :2, :3, :4, :5, :6, :7)`

	_, err := r.db.Exec(query,
		log.Transacao,
		log.Tabela,
		log.DataRecebimento,
		log.DataProcessamento,
		log.StatusProcessamento,
		log.JSON,
		log.DescricaoErro,
	)
	if err != nil {
		return fmt.Errorf("error saving log integration: %w", err)
	}

	return nil
}

// SendToQueue sends a message to queue (placeholder implementation)
func (r *ProductIntegrationRepository) SendToQueue(message entities.QueueMessage) error {
	// This would integrate with RabbitMQ or other message queue
	// For now, we'll just log the message
	messageJSON, _ := json.Marshal(message)
	log.Printf("Sending to queue: %s", string(messageJSON))
	return nil
}

// RemoverCaracteresEspeciais removes special characters from string
func (r *ProductIntegrationRepository) RemoverCaracteresEspeciais(input string) string {
	// Simple implementation - you may want to make this more sophisticated
	result := ""
	for _, char := range input {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') ||
			(char >= '0' && char <= '9') || char == ' ' {
			result += string(char)
		}
	}
	return result
}

// ValidateMarketingStructureLevel2 validates marketing structure level 2
func (r *ProductIntegrationRepository) ValidateMarketingStructureLevel2(ms *entities.MarketingStructure) *entities.LogValidate {
	if ms == nil {
		return &entities.LogValidate{
			Success: false,
			Message: "Estrutura mercadológica nível 2 não encontrada",
		}
	}

	if ms.IdNivelPai == nil || *ms.IdNivelPai <= 0 {
		return &entities.LogValidate{
			Success: false,
			Message: "Estrutura mercadológica nível 2 deve ter um nível pai válido",
		}
	}

	return &entities.LogValidate{
		Success: true,
		Message: "Estrutura mercadológica válida",
	}
}

// ValidateBrandDesc validates brand description
func (r *ProductIntegrationRepository) ValidateBrandDesc(descMarca string) *entities.LogValidate {
	if descMarca == "" {
		return &entities.LogValidate{
			Success: false,
			Message: "Descrição da marca não pode ser vazia",
		}
	}

	return &entities.LogValidate{
		Success: true,
		Message: "Marca válida",
	}
}

// AddOrUpdateStaging adds or updates a product integration staging record
func (r *ProductIntegrationRepository) AddOrUpdateStaging(idProduto, idDealer int, jsonPayload string) error {
	query := `
		MERGE INTO INTEGRACAOPRODUTOSTAGING dest
		USING (SELECT :idProduto AS IDPRODUTO, :idDealer AS IDREVENDEDOR FROM dual) src
		ON (dest.IDPRODUTO = src.IDPRODUTO AND dest.IDREVENDEDOR = src.IDREVENDEDOR)
		WHEN MATCHED THEN
			UPDATE SET dest.JSON = :jsonPayload, dest.DATAATUALIZACAO = SYSDATE
		WHEN NOT MATCHED THEN
			INSERT (IDPRODUTO, IDREVENDEDOR, JSON, DATAATUALIZACAO)
			VALUES (:idProduto, :idDealer, :jsonPayload, SYSDATE)
	`
	_, err := r.db.Exec(query, idProduto, idDealer, jsonPayload)
	if err != nil {
		return fmt.Errorf("error adding or updating product integration staging: %w", err)
	}
	return nil
}

// ValidateIndustry validates industry
func (r *ProductIntegrationRepository) ValidateIndustry(industry string) *entities.LogValidate {
	if industry == "" {
		return &entities.LogValidate{
			Success: false,
			Message: "Indústria não pode ser vazia",
		}
	}

	return &entities.LogValidate{
		Success: true,
		Message: "Indústria válida",
	}
}

// InsertProduct inserts a new product into the database
func (r *ProductIntegrationRepository) InsertProduct(product *entities.ProductNew) (int, error) {
	query := `INSERT INTO PRODUTO (
		DESCRICAO_PRODUTO, DESCRICAO_CUPOM, PITSTOP, ID_ESTRUTURA_MERCADOLOGICA, 
		ID_NIVEL1_ESTR_MERC, ID_NIVEL2_ESTR_MERC, ID_NIVEL3_ESTR_MERC, NOTABILIDADE, 
		CODIGO_RMS, ATIVO, MARKUP, PERIODO_SHELF_LIFE, SHELF_LIFE, TIPO_PRODUTO, PRODUCAO, 
		PRODU_DATA_ULTIMA_ATUALIZACAO, ID_MARCA, CONTEUDO_EMBALAGEM, FORA_MIX, REGIONAL, 
		ID_UNIDADE_MEDIDA, DIRETORIO_ANEXO, GIFT, OBSERVACAO, REFERENCIA_FABRICANTE
	) VALUES (
		:1, :2, :3, :4, :5, :6, :7, :8, :9, :10, :11, :12, :13, :14, :15, :16, :17, :18, :19, :20, :21, :22, :23, :24, :25
	) RETURNING ID_PRODUTO INTO :26`

	var newProductID int
	_, err := r.db.Exec(query,
		product.DescricaoProduto, product.DescricaoCupom, product.PitStop, product.IdEstruturaMercadologica,
		product.IdNivel1EstrMerc, product.IdNivel2EstrMerc, product.IdNivel3EstrMerc, product.Notabilidade,
		product.CodigoRMS, product.Ativo, product.MarkUp, product.PeriodoShelfLife, product.ShelfLife, product.TipoProduto, product.Producao,
		product.DataUltimaAtualizacao, product.IdMarca, product.ConteudoEmbalagem, product.ForaMix, product.Regional,
		product.IdUnidadeMedida, product.DiretorioAnexo, product.Gift, product.Observacao, product.ReferenciaFabricante,
		sql.Out{Dest: &newProductID},
	)
	if err != nil {
		return 0, fmt.Errorf("error inserting product: %w", err)
	}

	// Insert packaging
	for _, pkg := range product.Embalagens {
		err := r.InsertProductPackaging(newProductID, pkg)
		if err != nil {
			return 0, fmt.Errorf("error inserting product packaging for new product %d: %w", newProductID, err)
		}
	}

	return newProductID, nil
}

// UpdateProduct updates an existing product in the database
func (r *ProductIntegrationRepository) UpdateProduct(product *entities.ProductNew) error {
	query := `UPDATE PRODUTO SET
		DESCRICAO_PRODUTO = :1, DESCRICAO_CUPOM = :2, PITSTOP = :3, ID_ESTRUTURA_MERCADOLOGICA = :4, 
		ID_NIVEL1_ESTR_MERC = :5, ID_NIVEL2_ESTR_MERC = :6, ID_NIVEL3_ESTR_MERC = :7, NOTABILIDADE = :8, 
		CODIGO_RMS = :9, ATIVO = :10, MARKUP = :11, PERIODO_SHELF_LIFE = :12, SHELF_LIFE = :13, TIPO_PRODUTO = :14, PRODUCAO = :15, 
		PRODU_DATA_ULTIMA_ATUALIZACAO = :16, ID_MARCA = :17, CONTEUDO_EMBALAGEM = :18, FORA_MIX = :19, REGIONAL = :20, 
		ID_UNIDADE_MEDIDA = :21, DIRETORIO_ANEXO = :22, GIFT = :23, OBSERVACAO = :24, REFERENCIA_FABRICANTE = :25
	WHERE ID_PRODUTO = :26`

	_, err := r.db.Exec(query,
		product.DescricaoProduto, product.DescricaoCupom, product.PitStop, product.IdEstruturaMercadologica,
		product.IdNivel1EstrMerc, product.IdNivel2EstrMerc, product.IdNivel3EstrMerc, product.Notabilidade,
		product.CodigoRMS, product.Ativo, product.MarkUp, product.PeriodoShelfLife, product.ShelfLife, product.TipoProduto, product.Producao,
		product.DataUltimaAtualizacao, product.IdMarca, product.ConteudoEmbalagem, product.ForaMix, product.Regional,
		product.IdUnidadeMedida, product.DiretorioAnexo, product.Gift, product.Observacao, product.ReferenciaFabricante,
		product.IdProduto,
	)
	if err != nil {
		return fmt.Errorf("error updating product %d: %w", *product.IdProduto, err)
	}

	// Update packaging: delete existing and insert new ones
	err = r.DeleteProductPackagingByProductID(*product.IdProduto)
	if err != nil {
		return fmt.Errorf("error deleting old packaging for product %d: %w", *product.IdProduto, err)
	}

	for _, pkg := range product.Embalagens {
		err := r.InsertProductPackaging(*product.IdProduto, pkg)
		if err != nil {
			return fmt.Errorf("error inserting new packaging for product %d: %w", *product.IdProduto, err)
		}
	}

	return nil
}

// InsertProductPackaging inserts a new product packaging record
func (r *ProductIntegrationRepository) InsertProductPackaging(productID int, pkg entities.ProductPackaging) error {
	query := `INSERT INTO EMBALAGEM_PRODUTO (
		ID_PRODUTO, CODIGO_BARRAS, PRINCIPAL, QUANTIDADE_EMBALAGEM, ID_UNIDADE_MEDIDA, TIPO_CODIGO_BARRAS
	) VALUES (
		:1, :2, :3, :4, :5, :6
	)`
	_, err := r.db.Exec(query,
		productID, pkg.CodigoBarras, pkg.Principal, pkg.QuantidadeEmbalagem, pkg.IdUnidadeMedida, pkg.TipoCodigoBarras,
	)
	if err != nil {
		return fmt.Errorf("error inserting product packaging: %w", err)
	}
	return nil
}

// DeleteProductPackagingByProductID deletes all packaging records for a given product ID
func (r *ProductIntegrationRepository) DeleteProductPackagingByProductID(productID int) error {
	query := `DELETE FROM EMBALAGEM_PRODUTO WHERE ID_PRODUTO = :1`
	_, err := r.db.Exec(query, productID)
	if err != nil {
		return fmt.Errorf("error deleting product packaging by product ID %d: %w", productID, err)
	}
	return nil
}

// RemoveProductIntegrationStagingRecord removes a product integration staging record by ID
func (r *ProductIntegrationRepository) RemoveProductIntegrationStagingRecord(id int) error {
	query := `DELETE FROM INTEGRACAOPRODUTOSTAGING WHERE IDINTEGRACAOPRODUTOSTAGING = :1`
	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("error removing product integration staging record %d: %w", id, err)
	}
	return nil
}

// GetAllProductIntegrationStagingRecords retrieves all records from INTEGRACAOPRODUTOSTAGING
func (r *ProductIntegrationRepository) GetAllProductIntegrationStagingRecords() ([]entities.ProductIntegrationStaging, error) {
	query := `SELECT IDINTEGRACAOPRODUTOSTAGING, IDPRODUTO, IDREVENDEDOR, JSON, DATAATUALIZACAO
			  FROM INTEGRACAOPRODUTOSTAGING
			  ORDER BY IDINTEGRACAOPRODUTOSTAGING ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying product integration staging records: %w", err)
	}
	defer rows.Close()

	var results []entities.ProductIntegrationStaging
	for rows.Next() {
		var record entities.ProductIntegrationStaging
		err := rows.Scan(
			&record.IdIntegrationProdutoStaging,
			&record.IdProduto,
			&record.IdRevendedor,
			&record.Json,
			&record.DataAtualizacao,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning product integration staging record: %w", err)
		}
		results = append(results, record)
	}

	return results, nil
}
