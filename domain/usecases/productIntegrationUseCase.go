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
	repo             *repositories.ProductIntegrationRepository
	packagingUseCase *PackagingIntegrationUseCase
	db               *sql.DB
}

// NewProductIntegrationUseCase creates a new instance of ProductIntegrationUseCase
func NewProductIntegrationUseCase(repo *repositories.ProductIntegrationRepository, packagingUseCase *PackagingIntegrationUseCase, db *sql.DB) *ProductIntegrationUseCase {
	return &ProductIntegrationUseCase{
		repo:             repo,
		packagingUseCase: packagingUseCase,
		db:               db,
	}
}

// IntegrateProductService integrates a product and its packaging
func (uc *ProductIntegrationUseCase) IntegrateProductService(idProduto, idDealer int, item entities.ProductSelectIntegration) error {
	jsonPayload, err := json.Marshal(item)
	if err != nil {
		return fmt.Errorf("error marshalling product select: %w", err)
	}

	err = uc.repo.AddOrUpdateStaging(idProduto, idDealer, string(jsonPayload))
	if err != nil {
		return fmt.Errorf("error adding or updating product staging: %w", err)
	}

	err = uc.packagingUseCase.PackagingIntegrateService(idProduto, idDealer)
	if err != nil {
		return fmt.Errorf("error integrating packaging: %w", err)
	}

	return nil
}

// ImportProductIntegration is the main function that imports product integrations
func (uc *ProductIntegrationUseCase) ImportProductIntegration() (bool, error) {
	log.Println("Starting product integration import process")

	var success []bool

	// Get products from STAGING table, not from input table
	productsStagingRecords, err := uc.repo.GetAllProductIntegrationStagingRecords()
	if err != nil {
		return false, fmt.Errorf("error getting products from staging: %w", err)
	}

	log.Printf("Found %d product(s) to process from INTEGRACAOPRODUTOSTAGING", len(productsStagingRecords))

	if len(productsStagingRecords) == 0 {
		log.Println("No products found to process in staging table. Exiting.")
		return true, nil
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

	for i, stagingRecord := range productsStagingRecords {
		// Wrap each product processing in a func to catch panics
		func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("PANIC recovered while processing product %d/%d (Staging ID: %d): %v", 
						i+1, len(productsStagingRecords), stagingRecord.IdIntegrationProdutoStaging, r)
					success = append(success, false)
					
					// Log panic error
					logErro := entities.QueueMessage{
						Tabela: "LogIntegrRMS",
						Fields: []string{"TRANSACAO", "TABELA", "DATARECEBIMENTO", "DATAPROCESSAMENTO", "STATUSPROCESSAMENTO", "JSON", "DESCRICAOERRO"},
						Values: []interface{}{
							"STAGING",
							"PRODUTOS",
							stagingRecord.DataAtualizacao,
							time.Now(),
							0, // Status 0 = error
							stagingRecord.Json,
							fmt.Sprintf("PANIC: %v", r),
						},
					}
					uc.repo.SendToQueue(logErro)
				}
			}()

			log.Printf("Processing product %d/%d (Staging ID: %d, Product ID: %d, Dealer ID: %d)",
				i+1, len(productsStagingRecords),
				stagingRecord.IdIntegrationProdutoStaging,
				stagingRecord.IdProduto,
				stagingRecord.IdRevendedor)

			result := uc.processProductFromStaging(stagingRecord)
			log.Printf("Product %d processing result - Success: %v, Message: %s", i+1, result.Success, result.Message)

			logErro := entities.QueueMessage{
				Tabela: "LogIntegrRMS",
				Fields: []string{"TRANSACAO", "TABELA", "DATARECEBIMENTO", "DATAPROCESSAMENTO", "STATUSPROCESSAMENTO", "JSON", "DESCRICAOERRO"},
				Values: []interface{}{
					"STAGING",
					"PRODUTOS",
					stagingRecord.DataAtualizacao,
					time.Now(),
					uc.getStatusFromResult(result),
					stagingRecord.Json,
					uc.getMessageFromResult(result),
				},
			}

			// Send to queue (logging mechanism)
			if err := uc.repo.SendToQueue(logErro); err != nil {
				log.Printf("Error sending log to queue: %v", err)
			}

			if result.Success {
				success = append(success, true)
				log.Printf("✅ Product %d processed successfully from INTEGRACAOPRODUTOSTAGING", i+1)
			} else {
				success = append(success, false)
				log.Printf("❌ Product %d processing FAILED: %s", i+1, result.Message)
				log.Printf("⏩ Continuing to next product...")
			}
		}()
	}

	if err := tx.Commit(); err != nil {
		return false, fmt.Errorf("error committing transaction: %w", err)
	}

	// Calculate success/failure summary
	totalProducts := len(productsStagingRecords)
	successCount := 0
	failureCount := 0
	
	for _, val := range success {
		if val {
			successCount++
		} else {
			failureCount++
		}
	}

	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("📊 PRODUCT INTEGRATION SUMMARY")
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("   Total Products:     %d", totalProducts)
	log.Printf("   ✅ Successful:      %d (%.1f%%)", successCount, float64(successCount)/float64(totalProducts)*100)
	log.Printf("   ❌ Failed:          %d (%.1f%%)", failureCount, float64(failureCount)/float64(totalProducts)*100)
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Return true if at least one product succeeded
	return successCount > 0, nil
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
		log.Printf("ERROR: Failed to parse JSON for IPR_ID %v: %v", rms.IprID, err)
		return &entities.LogValidate{
			Success: false,
			Message: fmt.Sprintf("Error parsing JSON: %v", err),
		}
	}

	log.Printf("JSON parsed successfully for IPR_ID %v, calling Oracle procedure pkg_integra_produto.prc_integra_hermes", rms.IprID)

	// Call Oracle stored procedure to handle the integration
	if rms.IprID != nil {
		result, err := uc.repo.DoPackageProductIntegration(*rms.IprID)
		if err != nil {
			log.Printf("ERROR: Oracle procedure failed for IPR_ID %v: %v", rms.IprID, err)
			return &entities.LogValidate{
				Success: false,
				Message: fmt.Sprintf("Error executing Oracle procedure: %v", err),
			}
		}
		log.Printf("Oracle procedure completed for IPR_ID %v - Success: %v, Message: %s", rms.IprID, result.Success, result.Message)
		return result
	}

	log.Printf("ERROR: Invalid IPR_ID (nil) for product")
	return &entities.LogValidate{
		Success: false,
		Message: "Invalid IPR_ID",
	}
}

// processProductFromStaging processes a single product from staging table
func (uc *ProductIntegrationUseCase) processProductFromStaging(stagingRecord entities.ProductIntegrationStaging) *entities.LogValidate {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in processProductFromStaging: %v", r)
		}
	}()

	log.Printf("Processing staging record - ID: %d, Product ID: %d, Dealer ID: %d",
		stagingRecord.IdIntegrationProdutoStaging,
		stagingRecord.IdProduto,
		stagingRecord.IdRevendedor)

	// Parse JSON from staging into ProductSelectIntegration
	var productSelect entities.ProductSelectIntegration
	if err := json.Unmarshal([]byte(stagingRecord.Json), &productSelect); err != nil {
		log.Printf("ERROR: Failed to parse JSON from staging ID %d into ProductSelectIntegration: %v", stagingRecord.IdIntegrationProdutoStaging, err)
		return &entities.LogValidate{
			Success: false,
			Message: fmt.Sprintf("Error parsing JSON from staging: %v", err),
		}
	}

	// Construct ProductInJson from ProductSelectIntegration and other staging data
	// This mirrors the structure expected by getNewProduct
	productInJson := entities.ProductInJson{
		ProdutosSelect: []entities.ProductSelectIntegration{productSelect},
		Pesavel:        productSelect.Pesavel, // Assuming pesavel is part of ProductSelectIntegration or derivable
	}

	log.Printf("JSON parsed successfully from staging, calling getNewProduct for Product ID: %d", stagingRecord.IdProduto)

	// Call getNewProduct to process the product
	validationResult, err := uc.getNewProduct(productInJson)
	if err != nil {
		log.Printf("ERROR: getNewProduct failed for staging ID %d: %v", stagingRecord.IdIntegrationProdutoStaging, err)
		return &entities.LogValidate{
			Success: false,
			Message: fmt.Sprintf("Error processing product with getNewProduct: %v", err),
		}
	}

	if validationResult.Success {
		log.Printf("Product from staging processed successfully - Product ID: %d, Dealer ID: %d. Removing from staging.", stagingRecord.IdProduto, stagingRecord.IdRevendedor)
		// Remove from staging after successful processing
		err := uc.repo.RemoveProductIntegrationStagingRecord(stagingRecord.IdIntegrationProdutoStaging)
		if err != nil {
			log.Printf("ERROR: Failed to remove staging record %d: %v", stagingRecord.IdIntegrationProdutoStaging, err)
			// Log the error but still return success for the product processing itself
			return &entities.LogValidate{
				Success: true,
				Message: fmt.Sprintf("Product processed, but failed to remove from staging: %v", err),
			}
		}
	}

	return validationResult
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
	if produtoSelect.Subclasse.String() != "" {
		if val, err := strconv.Atoi(produtoSelect.Subclasse.String()); err == nil {
			newProduct.IdEstruturaMercadologica = &val
		}
	}

	if produtoSelect.Nivel1.String() != "" {
		if val, err := strconv.Atoi(produtoSelect.Nivel1.String()); err == nil {
			newProduct.IdNivel1EstrMerc = &val
		}
	}

	if produtoSelect.Depto.String() != "" {
		if val, err := strconv.Atoi(produtoSelect.Depto.String()); err == nil {
			newProduct.IdNivel2EstrMerc = &val
		}
	}

	// Set RMS Code - try both cod and codrms fields
	codigoRMS := produtoSelect.CodRMS.String()
	if codigoRMS == "" {
		codigoRMS = produtoSelect.Cod.String()
	}
	if codigoRMS != "" {
		if val, err := strconv.Atoi(codigoRMS); err == nil {
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

	var productID int
	// Check if product exists by RMS Code
	existingProduct, err := uc.repo.GetProductByCodeRMS(*newProduct.CodigoRMS)
	if err != nil {
		return fmt.Errorf("error checking existing product by RMS code: %w", err)
	}

	var codigoBarrasPrinc string
	for _, embalagem := range newProduct.Embalagens {
		if embalagem.Principal {
			codigoBarrasPrinc = embalagem.CodigoBarras
			break
		}
	}

	// If product doesn't exist by RMS code, check by primary barcode
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
			// Insert new product
			log.Printf("Inserting new product with RMS code: %d", *newProduct.CodigoRMS)
			newProductID, err := uc.repo.InsertProduct(newProduct)
			if err != nil {
				return fmt.Errorf("error inserting new product: %w", err)
			}
			productID = newProductID
		} else {
			// Update existing product
			newProduct.IdProduto = existingProduct.IdProduto
			log.Printf("Updating existing product with ID: %d", *newProduct.IdProduto)
			err := uc.repo.UpdateProduct(newProduct)
			if err != nil {
				return fmt.Errorf("error updating product %d: %w", *newProduct.IdProduto, err)
			}
			productID = *newProduct.IdProduto
		}
	} else {
		return fmt.Errorf("ID da marca é obrigatório")
	}

	// After successful insert/update, ensure DiretorioAnexo is set correctly
	directorioAnexo := fmt.Sprintf("P%d", productID)
	newProduct.DiretorioAnexo = directorioAnexo

	// Re-update the product to set DiretorioAnexo if it was just inserted
	if existingProduct == nil {
		err := uc.repo.UpdateProduct(newProduct)
		if err != nil {
			return fmt.Errorf("error updating product directory annex for product %d: %w", productID, err)
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
