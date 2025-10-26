package usecases

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/thiagohmm/integracaocron/domain/entities"
	"github.com/thiagohmm/integracaocron/domain/repositories"
)

// ProductIntegrationUseCase handles product integration business logic
type ProductIntegrationUseCase struct {
	repo *repositories.ProductIntegrationRepository
	db   *sql.DB
}

// NewProductIntegrationUseCase creates a new instance of ProductIntegrationUseCase
func NewProductIntegrationUseCase(repo *repositories.ProductIntegrationRepository, db *sql.DB) *ProductIntegrationUseCase {
	return &ProductIntegrationUseCase{
		repo: repo,
		db:   db,
	}
}

// ImportProductIntegration is the main function that imports product integrations
func (uc *ProductIntegrationUseCase) ImportProductIntegration() (bool, error) {
	log.Println("Starting product integration import process")

	var success []bool
	integrRmsProductsIn, err := uc.repo.GetIntegrRmsProductsIn()
	if err != nil {
		return false, fmt.Errorf("error getting integr rms products: %w", err)
	}

	// Begin transaction
	tx, err := uc.db.Begin()
	if err != nil {
		return false, fmt.Errorf("error starting transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	for _, rms := range integrRmsProductsIn {
		result := uc.processProductIntegration(rms)

		logErro := entities.QueueMessage{
			Tabela: "LogIntegrRMS",
			Fields: []string{"TRANSACAO", "TABELA", "DATARECEBIMENTO", "DATAPROCESSAMENTO", "STATUSPROCESSAMENTO", "JSON", "DESCRICAOERRO"},
			Values: []interface{}{
				"IN",
				"PRODUTOS",
				rms.DataRecebimento,
				time.Now(),
				uc.getStatusFromResult(result),
				uc.marshalRMS(rms),
				uc.getMessageFromResult(result),
			},
		}

		// Send to queue (logging mechanism)
		if err := uc.repo.SendToQueue(logErro); err != nil {
			log.Printf("Error sending log to queue: %v", err)
		}

		if result.Success {
			success = append(success, true)
			if err := uc.repo.RemoveProductService(rms); err != nil {
				log.Printf("Error removing product service: %v", err)
				success = append(success, false)
			}
		} else {
			success = append(success, false)
			// Still remove the record even if processing failed to avoid infinite loops
			if err := uc.repo.RemoveProductService(rms); err != nil {
				log.Printf("Error removing failed product service: %v", err)
			}
		}
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("error committing transaction: %w", err)
	}

	// Check if any processing failed
	isFalse := false
	for _, val := range success {
		if !val {
			isFalse = true
			break
		}
	}

	return !isFalse, nil
}

// processProductIntegration processes a single product integration
func (uc *ProductIntegrationUseCase) processProductIntegration(rms entities.IntegrRmsProductIn) *entities.LogValidate {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in processProductIntegration: %v", r)
		}
	}()

	// Parse JSON
	var produto entities.ProductInJson
	if err := json.Unmarshal([]byte(rms.JSON), &produto); err != nil {
		return &entities.LogValidate{
			Success: false,
			Message: fmt.Sprintf("Error parsing JSON: %v", err),
		}
	}

	// Call Oracle stored procedure to handle the integration
	if rms.IprID != nil {
		result, err := uc.repo.DoPackageProductIntegration(*rms.IprID)
		if err != nil {
			return &entities.LogValidate{
				Success: false,
				Message: fmt.Sprintf("Error executing Oracle procedure: %v", err),
			}
		}
		return result
	}

	return &entities.LogValidate{
		Success: false,
		Message: "Invalid IPR_ID",
	}
}

// getNewProduct processes and validates product data (commented out equivalent to TypeScript version)
func (uc *ProductIntegrationUseCase) getNewProduct(produto entities.ProductInJson) (*entities.LogValidate, error) {
	if len(produto.ProdutosSelect) == 0 {
		return &entities.LogValidate{
			Message: "Produto inválido ou vazio.",
			Success: false,
		}, nil
	}

	for _, produtoSelect := range produto.ProdutosSelect {
		newProduct, err := uc.createNewProductFromSelect(produtoSelect, produto.Pesavel)
		if err != nil {
			return &entities.LogValidate{
				Message: fmt.Sprintf("Error creating new product: %v", err),
				Success: false,
			}, nil
		}

		// Validate RMS Code
		if newProduct.CodigoRMS == nil || *newProduct.CodigoRMS <= 0 {
			return &entities.LogValidate{
				Message: "Código RMS deve ser maior que 0.",
				Success: false,
			}, nil
		}

		// Set default values
		uc.setProductDefaults(newProduct)

		// Validate marketing structure
		if validationResult := uc.validateMarketingStructure(newProduct); !validationResult.Success {
			return validationResult, nil
		}

		// Validate brand and industry
		if validationResult := uc.validateBrandAndIndustry(produtoSelect); !validationResult.Success {
			return validationResult, nil
		}

		// Process brand
		if err := uc.processBrand(newProduct, produtoSelect); err != nil {
			return &entities.LogValidate{
				Message: fmt.Sprintf("Error processing brand: %v", err),
				Success: false,
			}, nil
		}

		// Process barcodes and packaging
		uc.processBarcodesAndPackaging(newProduct, produtoSelect, produto.Pesavel)

		// Process product (insert or update)
		if err := uc.processProduct(newProduct); err != nil {
			return &entities.LogValidate{
				Message: fmt.Sprintf("Error processing product: %v", err),
				Success: false,
			}, nil
		}
	}

	produtoJSON, _ := json.Marshal(produto)
	return &entities.LogValidate{
		Message: fmt.Sprintf("Processamento realizado com sucesso. ProdutoIN: %s", string(produtoJSON)),
		Success: true,
	}, nil
}

// createNewProductFromSelect creates a ProductNew from ProductSelectIntegration
func (uc *ProductIntegrationUseCase) createNewProductFromSelect(produtoSelect entities.ProductSelectIntegration, pesavel string) (*entities.ProductNew, error) {
	newProduct := &entities.ProductNew{
		DescricaoProduto: produtoSelect.Desc,
		DescricaoCupom:   produtoSelect.DescEcf,
		Notabilidade:     entities.NOTABILIDADE,
	}

	// Set PitStop
	if produtoSelect.PitStop == entities.CONST_TRUE {
		newProduct.PitStop = 1
	} else {
		newProduct.PitStop = 0
	}

	// Set structure IDs
	if produtoSelect.Subclasse != "" {
		if val, err := strconv.Atoi(produtoSelect.Subclasse); err == nil {
			newProduct.IdEstruturaMercadologica = &val
		}
	}

	if produtoSelect.Nivel1 != "" {
		if val, err := strconv.Atoi(produtoSelect.Nivel1); err == nil {
			newProduct.IdNivel1EstrMerc = &val
		}
	}

	if produtoSelect.Depto != "" {
		if val, err := strconv.Atoi(produtoSelect.Depto); err == nil {
			newProduct.IdNivel2EstrMerc = &val
		}
	}

	// Set RMS Code
	if produtoSelect.CodRMS != "" {
		if val, err := strconv.Atoi(produtoSelect.CodRMS); err == nil {
			newProduct.CodigoRMS = &val
		}
	}

	// Set Active status
	newProduct.Ativo = (produtoSelect.Status == entities.CONST_ATIVO_A)

	return newProduct, nil
}

// setProductDefaults sets default values for a product
func (uc *ProductIntegrationUseCase) setProductDefaults(product *entities.ProductNew) {
	markup := 1.0
	product.MarkUp = &markup
	product.PeriodoShelfLife = ""
	shelfLife := 1
	product.ShelfLife = &shelfLife
	tipoProduto := 1
	product.TipoProduto = &tipoProduto
	producao := 1
	product.Producao = &producao
	now := time.Now()
	product.DataUltimaAtualizacao = &now
	foraMix := 1
	product.ForaMix = &foraMix
	regional := 1
	product.Regional = &regional
	conteudo := 1
	product.ConteudoEmbalagem = &conteudo
}

// validateMarketingStructure validates marketing structure
func (uc *ProductIntegrationUseCase) validateMarketingStructure(product *entities.ProductNew) *entities.LogValidate {
	if product.IdNivel2EstrMerc == nil {
		return &entities.LogValidate{
			Message: "IdNivel2EstrMerc é obrigatório",
			Success: false,
		}
	}

	marketingStructure, err := uc.repo.GetMarketingStructureLevel2(*product.IdNivel2EstrMerc)
	if err != nil {
		return &entities.LogValidate{
			Message: fmt.Sprintf("Erro ao obter estrutura mercadológica: %v", err),
			Success: false,
		}
	}

	validationResult := uc.repo.ValidateMarketingStructureLevel2(marketingStructure)
	if !validationResult.Success {
		return validationResult
	}

	// Set parent level
	if marketingStructure != nil && marketingStructure.IdNivelPai != nil {
		product.IdNivel1EstrMerc = marketingStructure.IdNivelPai
	}

	// Get level 4 structure
	if product.IdEstruturaMercadologica != nil {
		marketingStructure4, err := uc.repo.GetMarketingStructureLevel4(*product.IdEstruturaMercadologica)
		if err == nil && len(marketingStructure4) > 0 && marketingStructure4[0].IdNivelPai != nil {
			product.IdNivel3EstrMerc = marketingStructure4[0].IdNivelPai
		}
	}

	return &entities.LogValidate{Success: true, Message: "Marketing structure validated"}
}

// validateBrandAndIndustry validates brand and industry
func (uc *ProductIntegrationUseCase) validateBrandAndIndustry(produtoSelect entities.ProductSelectIntegration) *entities.LogValidate {
	vldBrandDesc := uc.repo.ValidateBrandDesc(produtoSelect.DescMarca)
	if !vldBrandDesc.Success {
		return vldBrandDesc
	}

	vldIndustry := uc.repo.ValidateIndustry(produtoSelect.Ind)
	if !vldIndustry.Success {
		return vldIndustry
	}

	return &entities.LogValidate{Success: true, Message: "Brand and industry validated"}
}

// processBrand processes brand information
func (uc *ProductIntegrationUseCase) processBrand(newProduct *entities.ProductNew, produtoSelect entities.ProductSelectIntegration) error {
	// Get existing brand
	brands, err := uc.repo.GetBrandByIndustryName(produtoSelect.DescMarca, produtoSelect.Ind)
	if err != nil {
		return fmt.Errorf("error getting brand: %w", err)
	}

	if len(brands) > 0 {
		// Brand exists
		newProduct.IdMarca = brands[len(brands)-1].IdMarca
	} else {
		// Create new brand
		status, _ := strconv.Atoi(produtoSelect.Status)
		industry, err := uc.repo.GetIndustryByNameAndStatus(produtoSelect.Ind, status)
		if err != nil {
			return fmt.Errorf("error getting industry: %w", err)
		}

		var industryResult *entities.Industry
		if industry == nil {
			// Create new industry
			newIndustry := entities.Industry{
				NomeIndustria:   produtoSelect.Ind,
				StatusIndustria: 1,
			}
			industryResult, err = uc.repo.SaveIndustry(newIndustry)
			if err != nil {
				return fmt.Errorf("error saving industry: %w", err)
			}
		} else {
			industryResult = industry
		}

		// Create new brand
		newBrand := entities.Brand{
			IdIndustria:   industryResult.IdIndustria,
			NomeMarca:     produtoSelect.DescMarca,
			StatusMarca:   1,
			NomeIndustria: industryResult.NomeIndustria,
		}

		brandResult, err := uc.repo.SaveBrand(newBrand)
		if err != nil {
			return fmt.Errorf("error saving brand: %w", err)
		}

		newProduct.IdMarca = brandResult.IdMarca
	}

	return nil
}

// processBarcodesAndPackaging processes barcodes and packaging
func (uc *ProductIntegrationUseCase) processBarcodesAndPackaging(newProduct *entities.ProductNew, produtoSelect entities.ProductSelectIntegration, pesavel string) {
	var tipoCodigoBarras string

	if len(produtoSelect.CodBarras) > 0 {
		newProduct.Embalagens = []entities.ProductPackaging{}

		for _, cbarra := range produtoSelect.CodBarras {
			unidadeMedida := entities.UNIDADE_MEDIDA_UN
			if pesavel == entities.CONST_TRUE {
				unidadeMedida = entities.UNIDADE_MEDIDA_KG
			}

			tipoCodigoBarra := entities.CB_INTERNO
			if cbarra.Tipo == entities.CB_BARRA_EAN13 {
				tipoCodigoBarra = entities.CB_BARRA_EAN
			}

			productPackaging := entities.ProductPackaging{
				CodigoBarras:        cbarra.CBarra,
				Principal:           (cbarra.Princ == entities.CONST_TRUE),
				QuantidadeEmbalagem: 1,
				IdUnidadeMedida:     &unidadeMedida,
				TipoCodigoBarras:    tipoCodigoBarra,
			}

			newProduct.IdUnidadeMedida = productPackaging.IdUnidadeMedida

			if tipoCodigoBarras == "" {
				tipoCodigoBarras = productPackaging.TipoCodigoBarras
			}

			newProduct.Embalagens = append(newProduct.Embalagens, productPackaging)
		}

		// Process additional packaging
		for _, embalagem := range produtoSelect.Embalagem {
			qtde := 0
			if embalagem.Qtde != "" {
				qtde, _ = strconv.Atoi(embalagem.Qtde)
			}

			unidadeMedida := entities.UNIDADE_MEDIDA_UN
			if produtoSelect.Pesavel == entities.CONST_TRUE {
				unidadeMedida = entities.UNIDADE_MEDIDA_KG
			}

			emb := entities.ProductPackaging{
				CodigoBarras:        embalagem.EAN,
				Principal:           false,
				QuantidadeEmbalagem: qtde,
				IdUnidadeMedida:     &unidadeMedida,
				TipoCodigoBarras:    tipoCodigoBarras,
			}

			newProduct.Embalagens = append(newProduct.Embalagens, emb)
		}
	}
}

// processProduct processes the product (insert or update)
func (uc *ProductIntegrationUseCase) processProduct(newProduct *entities.ProductNew) error {
	if newProduct.CodigoRMS == nil {
		return fmt.Errorf("código RMS é obrigatório")
	}

	// Check if product exists
	existingProduct, err := uc.repo.GetProductByCodeRMS(*newProduct.CodigoRMS)
	if err != nil {
		return fmt.Errorf("error checking existing product: %w", err)
	}

	var codigoBarrasPrinc string
	for _, embalagem := range newProduct.Embalagens {
		if embalagem.Principal {
			codigoBarrasPrinc = embalagem.CodigoBarras
			break
		}
	}

	// If product doesn't exist, check by barcode
	if existingProduct == nil && codigoBarrasPrinc != "" {
		embProduct, err := uc.repo.GetProductPackagingByBarCode(codigoBarrasPrinc)
		if err != nil {
			return fmt.Errorf("error getting product packaging by barcode: %w", err)
		}

		if embProduct != nil && embProduct.IdProduto != nil {
			existingProduct, err = uc.repo.GetProductByCodeRMS(*embProduct.IdProduto)
			if err != nil {
				return fmt.Errorf("error getting product by packaging ID: %w", err)
			}
		}
	}

	if newProduct.IdMarca != nil {
		if existingProduct == nil {
			// Insert new product - this would need actual implementation
			log.Printf("Would insert new product with RMS code: %d", *newProduct.CodigoRMS)
		} else {
			// Update existing product - this would need actual implementation
			newProduct.IdProduto = existingProduct.IdProduto
			log.Printf("Would update existing product with ID: %d", *newProduct.IdProduto)
		}
	}

	return nil
}

// Helper functions
func (uc *ProductIntegrationUseCase) getStatusFromResult(result *entities.LogValidate) int {
	if result.Success {
		return 0
	}
	return 1
}

func (uc *ProductIntegrationUseCase) getMessageFromResult(result *entities.LogValidate) string {
	if result.Success {
		return "Integração de Produtos Realizada com Sucesso"
	}
	return result.Message
}

func (uc *ProductIntegrationUseCase) marshalRMS(rms entities.IntegrRmsProductIn) string {
	data, _ := json.Marshal(rms)
	return string(data)
}
