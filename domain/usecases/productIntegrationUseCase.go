package usecases

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/thiagohmm/integracaocron/domain/entities"
	"github.com/thiagohmm/integracaocron/domain/repositories"
	"github.com/thiagohmm/integracaocron/pkg/tracing"
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
	ctx := context.Background()
	ctx, span := tracing.StartSpan(ctx, "IntegrateProductService")
	defer span.End()

	// ✅ Adicionar idproduto para pesquisa no Jaeger
	tracing.AddProductID(ctx, idProduto)
	tracing.AddIntAttribute(ctx, "dealer.id", idDealer)

	jsonPayload, err := json.Marshal(item)
	if err != nil {
		tracing.RecordError(ctx, err)
		return fmt.Errorf("error marshalling product select: %w", err)
	}

	err = uc.repo.AddOrUpdateStaging(idProduto, idDealer, string(jsonPayload))
	if err != nil {
		tracing.RecordError(ctx, err)
		return fmt.Errorf("error adding or updating product staging: %w", err)
	}

	err = uc.packagingUseCase.PackagingIntegrateService(idProduto, idDealer)
	if err != nil {
		tracing.RecordError(ctx, err)
		return fmt.Errorf("error integrating packaging: %w", err)
	}

	return nil
}

// ImportProductIntegration is the main function that imports product integrations
// Segue o mesmo fluxo do TypeScript: busca de INTEGRRMSPRODUTOIN, chama dopkg_produto(IPR_ID), remove registro
func (uc *ProductIntegrationUseCase) ImportProductIntegration() (bool, error) {
	ctx := context.Background()
	ctx, span := tracing.StartSpan(ctx, "ImportProductIntegration")
	defer span.End()

	log.Println("Starting product integration import process")

	// ✅ FLUXO CORRETO: Buscar de INTEGRRMSPRODUTOIN (tabela de entrada) igual ao TypeScript
	integrRmsProductsIn, err := uc.repo.GetIntegrRmsProductsIn()
	if err != nil {
		tracing.RecordError(ctx, err)
		return false, fmt.Errorf("error getting products from INTEGRRMSPRODUTOIN: %w", err)
	}

	log.Printf("Found %d product(s) to process from INTEGRRMSPRODUTOIN", len(integrRmsProductsIn))
	tracing.AddIntAttribute(ctx, "total_products", len(integrRmsProductsIn))

	if len(integrRmsProductsIn) == 0 {
		log.Println("No products found to process in INTEGRRMSPRODUTOIN. Exiting.")
		return true, nil
	}

	// Begin transaction
	tx, err := uc.db.Begin()
	if err != nil {
		return false, fmt.Errorf("error starting transaction: %w", err)
	}

	// ✅ CORREÇÃO CRÍTICA: Garantir que transação SEMPRE seja fechada
	committed := false
	defer func() {
		if !committed {
			if err := tx.Rollback(); err != nil {
				log.Printf("Error rolling back transaction: %v", err)
			}
		}
	}()

	// ✅ CORREÇÃO: Usar contadores ao invés de array para economizar memória
	var successCount, failureCount int
	totalProducts := len(integrRmsProductsIn)

	// ✅ OTIMIZAÇÃO: Aumentar batch size para melhor throughput
	const batchSize = 200
	var processedIDs []*int // Coletar IDs processados para batch delete

	for batchStart := 0; batchStart < totalProducts; batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > totalProducts {
			batchEnd = totalProducts
		}

		batch := integrRmsProductsIn[batchStart:batchEnd]
		log.Printf("Processing batch %d-%d of %d products", batchStart+1, batchEnd, totalProducts)

		// 🔍 Tracing: Criar span para o batch
		batchCtx, batchSpan := tracing.StartSpan(ctx, "ProcessProductBatch")
		tracing.AddIntAttribute(batchCtx, "batch.start", batchStart+1)
		tracing.AddIntAttribute(batchCtx, "batch.end", batchEnd)
		tracing.AddIntAttribute(batchCtx, "batch.size", len(batch))

		for i, rms := range batch {
			actualIndex := batchStart + i + 1

			// 🔍 Tracing: Criar span para cada produto
			productCtx, productSpan := tracing.StartSpan(batchCtx, "ProcessSingleProduct")
			if rms.IprID != nil {
				tracing.AddInt64Attribute(productCtx, "ipr_id", int64(*rms.IprID))
			}
			tracing.AddIntAttribute(productCtx, "product_index", actualIndex)

			// ✅ Adicionar idproduto para pesquisa no Jaeger (extrair do JSON se disponível)
			if rms.JSON != "" {
				var produto entities.ProductInJson
				if err := json.Unmarshal([]byte(rms.JSON), &produto); err == nil {
					if len(produto.ProdutosSelect) > 0 {
						// Tentar extrair código RMS que pode ser usado como ID
						codRMS := produto.ProdutosSelect[0].CodRMS.String()
						if codRMS != "" {
							if cod, err := strconv.Atoi(codRMS); err == nil {
								tracing.AddProductID(productCtx, cod)
							}
						}
					}
				}
			}

			// ✅ FLUXO CORRETO: Processar igual ao TypeScript
			func() {
				defer productSpan.End()

				// ✅ CORREÇÃO: Garantir que o ID seja adicionado para remoção mesmo em caso de panic/erro
				// No TypeScript, removeProductService é chamado sempre (tanto sucesso quanto erro)
				defer func() {
					// Sempre adicionar ID para remoção (igual TypeScript)
					if rms.IprID != nil {
						processedIDs = append(processedIDs, rms.IprID)
					}
				}()

				defer func() {
					if r := recover(); r != nil {
						iprIDStr := "nil"
						if rms.IprID != nil {
							iprIDStr = fmt.Sprintf("%d", *rms.IprID)
						}
						log.Printf("PANIC recovered while processing product %d/%d (IPR_ID: %s): %v",
							actualIndex, totalProducts, iprIDStr, r)
						failureCount++

						// 🔍 Tracing: Registrar panic
						tracing.AddEvent(productCtx, "panic.recovered",
							tracing.StringAttr("panic_value", fmt.Sprintf("%v", r)),
							tracing.StringAttr("ipr_id", iprIDStr))
						tracing.SetStatus(productCtx, 2, fmt.Sprintf("PANIC: %v", r)) // codes.Error = 2

						// ✅ FLUXO CORRETO: Log panic error igual TypeScript catch
						// TypeScript usa JSON.stringify(rms) no catch também
						rmsJSON, _ := json.Marshal(rms)
						logErro := entities.QueueMessage{
							Tabela: "LogsIntegrRMS",
							Fields: []string{"TRANSACAO", "TABELA", "DATARECEBIMENTO", "DATAPROCESSAMENTO", "STATUSPROCESSAMENTO", "JSON", "DESCRICAOERRO"},
							Values: []interface{}{
								"IN",
								"PRODUTOS",
								rms.DataRecebimento,
								time.Now(),
								1,               // Status 1 = error (igual TypeScript)
								string(rmsJSON), // JSON completo do objeto rms (igual TypeScript: JSON.stringify(rms))
								fmt.Sprintf("PANIC: %v", r),
							},
						}
						uc.repo.SendToQueue(logErro)
					}
				}()

				iprIDLog := "nil"
				if rms.IprID != nil {
					iprIDLog = fmt.Sprintf("%d", *rms.IprID)
				}
				log.Printf("Processing product %d/%d (IPR_ID: %s)", actualIndex, totalProducts, iprIDLog)

				// ✅ FLUXO CORRETO: Chama processProductIntegration igual TypeScript
				result := uc.processProductIntegration(rms)
				log.Printf("Product %d processing result - Success: %v, Message: %s", actualIndex, result.Success, result.Message)

				// 🔍 Tracing: Registrar resultado do processamento
				tracing.AddBoolAttribute(productCtx, "processing.success", result.Success)
				tracing.AddStringAttribute(productCtx, "processing.message", result.Message)

				// ✅ FLUXO CORRETO: Log igual TypeScript
				// TypeScript usa JSON.stringify(rms) que serializa o objeto completo
				// No Go, serializamos o objeto rms completo para corresponder
				rmsJSON, _ := json.Marshal(rms)
				logErro := entities.QueueMessage{
					Tabela: "LogsIntegrRMS",
					Fields: []string{"TRANSACAO", "TABELA", "DATARECEBIMENTO", "DATAPROCESSAMENTO", "STATUSPROCESSAMENTO", "JSON", "DESCRICAOERRO"},
					Values: []interface{}{
						"IN",
						"PRODUTOS",
						rms.DataRecebimento,
						time.Now(),
						uc.getStatusFromResult(result), // 0 = sucesso, 1 = erro (igual TypeScript)
						string(rmsJSON),                // JSON completo do objeto rms (igual TypeScript: JSON.stringify(rms))
						uc.getMessageFromResult(result),
					},
				}

				// Send to queue (logging mechanism)
				if err := uc.repo.SendToQueue(logErro); err != nil {
					log.Printf("Error sending log to queue: %v", err)
					tracing.RecordError(productCtx, err)
				}

				// ✅ NOTA: O ID já é adicionado ao processedIDs no defer acima
				// Isso garante remoção mesmo em caso de panic/erro (igual TypeScript)

				// ✅ Incrementar contadores
				if result.Success {
					successCount++
					log.Printf("✅ Product %d processed successfully (IPR_ID: %s)", actualIndex, iprIDLog)
					tracing.SetStatus(productCtx, 1, "Success") // codes.Ok = 1
				} else {
					failureCount++
					log.Printf("❌ Product %d processing FAILED: %s (IPR_ID: %s)", actualIndex, result.Message, iprIDLog)
					log.Printf("⏩ Continuing to next product...")
					tracing.SetStatus(productCtx, 2, result.Message) // codes.Error = 2
				}
			}()
		}

		// 🔍 Tracing: Finalizar span do batch e registrar estatísticas
		tracing.AddIntAttribute(batchCtx, "batch.success_count", successCount)
		tracing.AddIntAttribute(batchCtx, "batch.failure_count", failureCount)
		batchSpan.End()

		// ✅ OTIMIZAÇÃO: Batch delete após cada batch de processamento
		if len(processedIDs) > 0 {
			if err := uc.repo.RemoveProductsServiceBatch(processedIDs); err != nil {
				log.Printf("Warning: Error in batch delete: %v", err)
			} else {
				log.Printf("Batch deleted %d processed records", len(processedIDs))
			}
			processedIDs = nil // Limpar para próximo batch
		}

		// ✅ CORREÇÃO: Liberar batch da memória após processar
		batch = nil

		// Log progresso do batch
		log.Printf("Batch completed. Current stats - Success: %d, Failed: %d", successCount, failureCount)
	}

	// ✅ CORREÇÃO: Commit com flag para evitar rollback no defer
	if err := tx.Commit(); err != nil {
		tracing.RecordError(ctx, err)
		return false, fmt.Errorf("error committing transaction: %w", err)
	}
	committed = true

	// 🔍 Tracing: Registrar estatísticas finais
	tracing.AddIntAttribute(ctx, "final.success_count", successCount)
	tracing.AddIntAttribute(ctx, "final.failure_count", failureCount)
	tracing.AddFloatAttribute(ctx, "final.success_rate", float64(successCount)/float64(totalProducts)*100)
	tracing.SetStatus(ctx, 1, "Product integration completed successfully") // codes.Ok = 1

	// Summary
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("📊 PRODUCT INTEGRATION SUMMARY")
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("   Total Products:     %d", totalProducts)
	log.Printf("   ✅ Successful:      %d (%.1f%%)", successCount, float64(successCount)/float64(totalProducts)*100)
	log.Printf("   ❌ Failed:          %d (%.1f%%)", failureCount, float64(failureCount)/float64(totalProducts)*100)
	log.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// ✅ CORREÇÃO: Retornar true se TODOS foram sucesso (igual TypeScript: !isFalse)
	// TypeScript: const isFalse: boolean = success.some((valor) => valor === false)
	// TypeScript: return !isFalse  // true se nenhum foi false
	// Go equivalente: true se nenhum falhou (failureCount == 0)
	allSucceeded := failureCount == 0
	return allSucceeded, nil
}

// processProductIntegration processes a single product integration
func (uc *ProductIntegrationUseCase) processProductIntegration(rms entities.IntegrRmsProductIn) *entities.LogValidate {
	ctx := context.Background()
	ctx, span := tracing.StartSpan(ctx, "ProcessProductIntegration")
	defer span.End()

	// 🔍 Tracing: Registrar IPR_ID
	if rms.IprID != nil {
		tracing.AddInt64Attribute(ctx, "ipr_id", int64(*rms.IprID))
	}

	// ✅ Adicionar idproduto para pesquisa no Jaeger
	if rms.JSON != "" {
		var produto entities.ProductInJson
		if err := json.Unmarshal([]byte(rms.JSON), &produto); err == nil {
			if len(produto.ProdutosSelect) > 0 {
				// Tentar extrair código RMS que pode ser usado como ID
				codRMS := produto.ProdutosSelect[0].CodRMS.String()
				if codRMS == "" {
					codRMS = produto.ProdutosSelect[0].Cod.String()
				}
				if codRMS != "" {
					if cod, err := strconv.Atoi(codRMS); err == nil {
						tracing.AddProductID(ctx, cod)
					}
				}
			}
		}
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in processProductIntegration: %v", r)
			tracing.AddEvent(ctx, "panic.recovered", tracing.StringAttr("panic_value", fmt.Sprintf("%v", r)))
		}
	}()

	// Validar se JSON não está vazio
	if strings.TrimSpace(rms.JSON) == "" {
		msg := "JSON vazio - não é possível processar"
		log.Printf("❌ ERROR: %s for IPR_ID %v", msg, rms.IprID)
		tracing.AddEvent(ctx, "validation.failed", tracing.StringAttr("reason", "empty_json"))
		tracing.SetStatus(ctx, 2, msg) // codes.Error = 2
		return &entities.LogValidate{
			Success: false,
			Message: msg,
		}
	}

	// Parse JSON
	var produto entities.ProductInJson
	if err := json.Unmarshal([]byte(rms.JSON), &produto); err != nil {
		log.Printf("❌ ERROR: Failed to parse JSON for IPR_ID %v: %v", rms.IprID, err)
		tracing.RecordError(ctx, err)
		tracing.AddEvent(ctx, "json.parse.failed")

		// ✅ Log no Jaeger: Erro ao fazer parse do JSON
		tracing.LogError(ctx, "Failed to parse product JSON",
			tracing.StringAttr("ipr_id", fmt.Sprintf("%v", rms.IprID)),
			tracing.StringAttr("json.original", rms.JSON),
			tracing.StringAttr("error", err.Error()))

		return &entities.LogValidate{
			Success: false,
			Message: fmt.Sprintf("Error parsing JSON: %v", err),
		}
	}

	log.Printf("✅ JSON parsed successfully for IPR_ID %v, calling Oracle procedure pkg_integra_produto.prc_integra_hermes", rms.IprID)
	tracing.AddEvent(ctx, "json.parsed.success")

	// ✅ Log no Jaeger: Produto antes de chamar dopkg_produto
	// Extrair informações do produto para o log
	var codRMS, descProduto string
	if len(produto.ProdutosSelect) > 0 {
		codRMS = produto.ProdutosSelect[0].CodRMS.String()
		if codRMS == "" {
			codRMS = produto.ProdutosSelect[0].Cod.String()
		}
		descProduto = produto.ProdutosSelect[0].Desc
	}

	// ✅ Log estruturado no Jaeger com todas as informações do produto
	tracing.LogInfo(ctx, "Calling dopkg_produto (Oracle procedure)",
		tracing.StringAttr("ipr_id", fmt.Sprintf("%v", rms.IprID)),
		tracing.StringAttr("json.original", rms.JSON),
		tracing.StringAttr("data_recebimento", rms.DataRecebimento.Format(time.RFC3339)),
		tracing.StringAttr("cod_rms", codRMS),
		tracing.StringAttr("desc_produto", descProduto),
		tracing.StringAttr("produtos_count", fmt.Sprintf("%d", len(produto.ProdutosSelect))),
		tracing.StringAttr("procedure", "pkg_integra_produto.prc_integra_hermes"))

	// Call Oracle stored procedure to handle the integration
	if rms.IprID != nil {
		result, err := uc.repo.DoPackageProductIntegration(*rms.IprID)
		if err != nil {
			log.Printf("❌ ERROR: Oracle procedure failed for IPR_ID %v: %v", rms.IprID, err)
			tracing.RecordError(ctx, err)
			tracing.SetStatus(ctx, 2, fmt.Sprintf("Oracle error: %v", err))

			// ✅ Log no Jaeger: Erro na procedure Oracle
			tracing.LogError(ctx, "Oracle procedure dopkg_produto failed",
				tracing.StringAttr("ipr_id", fmt.Sprintf("%d", *rms.IprID)),
				tracing.StringAttr("json.original", rms.JSON),
				tracing.StringAttr("cod_rms", codRMS),
				tracing.StringAttr("desc_produto", descProduto),
				tracing.StringAttr("procedure", "pkg_integra_produto.prc_integra_hermes"),
				tracing.StringAttr("error", err.Error()))

			return &entities.LogValidate{
				Success: false,
				Message: fmt.Sprintf("Error executing Oracle procedure: %v", err),
			}
		}
		log.Printf("Oracle procedure completed for IPR_ID %v - Success: %v, Message: %s", rms.IprID, result.Success, result.Message)

		// ✅ Log no Jaeger: Resultado da procedure Oracle
		if result.Success {
			tracing.LogInfo(ctx, "Oracle procedure dopkg_produto completed successfully",
				tracing.StringAttr("ipr_id", fmt.Sprintf("%d", *rms.IprID)),
				tracing.StringAttr("json.original", rms.JSON),
				tracing.StringAttr("cod_rms", codRMS),
				tracing.StringAttr("desc_produto", descProduto),
				tracing.StringAttr("procedure", "pkg_integra_produto.prc_integra_hermes"),
				tracing.StringAttr("result.message", result.Message))
		} else {
			tracing.LogError(ctx, "Oracle procedure dopkg_produto completed with error",
				tracing.StringAttr("ipr_id", fmt.Sprintf("%d", *rms.IprID)),
				tracing.StringAttr("json.original", rms.JSON),
				tracing.StringAttr("cod_rms", codRMS),
				tracing.StringAttr("desc_produto", descProduto),
				tracing.StringAttr("procedure", "pkg_integra_produto.prc_integra_hermes"),
				tracing.StringAttr("result.message", result.Message))
		}

		// 🔍 Tracing: Registrar resultado
		tracing.AddBoolAttribute(ctx, "oracle.success", result.Success)
		tracing.AddStringAttribute(ctx, "oracle.message", result.Message)

		if result.Success {
			tracing.SetStatus(ctx, 1, "Success") // codes.Ok = 1
		} else {
			tracing.SetStatus(ctx, 2, result.Message) // codes.Error = 2
		}

		return result
	}

	log.Printf("❌ ERROR: Invalid IPR_ID (nil) for product")
	tracing.AddEvent(ctx, "validation.failed", tracing.StringAttr("reason", "nil_ipr_id"))
	return &entities.LogValidate{
		Success: false,
		Message: "Invalid IPR_ID",
	}
}

// processProductFromStaging processes a single product from staging table
// This function calls the Oracle procedure pkg_integra_produto.prc_integra_hermes
// just like the TypeScript version does with dopkg_produto
func (uc *ProductIntegrationUseCase) processProductFromStaging(stagingRecord entities.ProductIntegrationStaging) *entities.LogValidate {
	ctx := context.Background()
	ctx, span := tracing.StartSpan(ctx, "ProcessProductFromStaging")
	defer span.End()

	// ✅ Tracing: Registrar IDs
	tracing.AddInt64Attribute(ctx, "staging_id", int64(stagingRecord.IdIntegrationProdutoStaging))
	tracing.AddInt64Attribute(ctx, "product_id", int64(stagingRecord.IdProduto))
	tracing.AddInt64Attribute(ctx, "dealer_id", int64(stagingRecord.IdRevendedor))
	tracing.AddProductID(ctx, int(stagingRecord.IdProduto))

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in processProductFromStaging: %v", r)
			tracing.LogError(ctx, "Panic recovered in processProductFromStaging",
				tracing.StringAttr("panic_value", fmt.Sprintf("%v", r)),
				tracing.IntAttr("staging_id", stagingRecord.IdIntegrationProdutoStaging),
				tracing.IntAttr("product_id", stagingRecord.IdProduto))
		}
	}()

	log.Printf("Processing staging record - Staging ID: %d, Product ID: %d, Dealer ID: %d",
		stagingRecord.IdIntegrationProdutoStaging,
		stagingRecord.IdProduto,
		stagingRecord.IdRevendedor)

	// Parse JSON to validate it (optional, but good for logging)
	var productSelect entities.ProductSelectIntegration
	var codRMS, descProduto string
	if err := json.Unmarshal([]byte(stagingRecord.Json), &productSelect); err != nil {
		log.Printf("ERROR: Failed to parse JSON from staging ID %d: %v", stagingRecord.IdIntegrationProdutoStaging, err)

		// ✅ Log no Jaeger: Erro ao fazer parse do JSON
		tracing.LogError(ctx, "Failed to parse JSON from staging",
			tracing.IntAttr("staging_id", stagingRecord.IdIntegrationProdutoStaging),
			tracing.IntAttr("product_id", stagingRecord.IdProduto),
			tracing.StringAttr("json.original", stagingRecord.Json),
			tracing.StringAttr("error", err.Error()))

		return &entities.LogValidate{
			Success: false,
			Message: fmt.Sprintf("Error parsing JSON from staging: %v", err),
		}
	} else {
		// Extrair informações do produto
		codRMS = productSelect.CodRMS.String()
		if codRMS == "" {
			codRMS = productSelect.Cod.String()
		}
		descProduto = productSelect.Desc
	}

	log.Printf("JSON parsed successfully from staging, calling Oracle procedure pkg_integra_produto.prc_integra_hermes for Product ID: %d", stagingRecord.IdProduto)

	// ✅ Log no Jaeger: Produto antes de chamar dopkg_produto (staging)
	tracing.LogInfo(ctx, "Calling dopkg_produto (Oracle procedure) from staging",
		tracing.IntAttr("staging_id", stagingRecord.IdIntegrationProdutoStaging),
		tracing.IntAttr("product_id", stagingRecord.IdProduto),
		tracing.IntAttr("dealer_id", stagingRecord.IdRevendedor),
		tracing.StringAttr("json.original", stagingRecord.Json),
		tracing.StringAttr("cod_rms", codRMS),
		tracing.StringAttr("desc_produto", descProduto),
		tracing.StringAttr("procedure", "pkg_integra_produto.prc_integra_hermes"))

	// Call Oracle procedure to process the product (just like TypeScript dopkg_produto)
	// The procedure receives the Product ID and processes the integration
	result, err := uc.repo.DoPackageProductIntegration(stagingRecord.IdProduto)
	if err != nil {
		log.Printf("ERROR: Oracle procedure failed for Product ID %d (Staging ID: %d): %v",
			stagingRecord.IdProduto, stagingRecord.IdIntegrationProdutoStaging, err)

		// ✅ Log no Jaeger: Erro na procedure Oracle (staging)
		tracing.LogError(ctx, "Oracle procedure dopkg_produto failed (from staging)",
			tracing.IntAttr("staging_id", stagingRecord.IdIntegrationProdutoStaging),
			tracing.IntAttr("product_id", stagingRecord.IdProduto),
			tracing.IntAttr("dealer_id", stagingRecord.IdRevendedor),
			tracing.StringAttr("json.original", stagingRecord.Json),
			tracing.StringAttr("cod_rms", codRMS),
			tracing.StringAttr("desc_produto", descProduto),
			tracing.StringAttr("procedure", "pkg_integra_produto.prc_integra_hermes"),
			tracing.StringAttr("error", err.Error()))

		return &entities.LogValidate{
			Success: false,
			Message: fmt.Sprintf("Error executing Oracle procedure: %v", err),
		}
	}

	log.Printf("Oracle procedure completed for Product ID %d - Success: %v, Message: %s",
		stagingRecord.IdProduto, result.Success, result.Message)

	// ✅ Log no Jaeger: Resultado da procedure Oracle (staging)
	if result.Success {
		tracing.LogInfo(ctx, "Oracle procedure dopkg_produto completed successfully (from staging)",
			tracing.IntAttr("staging_id", stagingRecord.IdIntegrationProdutoStaging),
			tracing.IntAttr("product_id", stagingRecord.IdProduto),
			tracing.IntAttr("dealer_id", stagingRecord.IdRevendedor),
			tracing.StringAttr("json.original", stagingRecord.Json),
			tracing.StringAttr("cod_rms", codRMS),
			tracing.StringAttr("desc_produto", descProduto),
			tracing.StringAttr("procedure", "pkg_integra_produto.prc_integra_hermes"),
			tracing.StringAttr("result.message", result.Message))
	} else {
		tracing.LogError(ctx, "Oracle procedure dopkg_produto completed with error (from staging)",
			tracing.IntAttr("staging_id", stagingRecord.IdIntegrationProdutoStaging),
			tracing.IntAttr("product_id", stagingRecord.IdProduto),
			tracing.IntAttr("dealer_id", stagingRecord.IdRevendedor),
			tracing.StringAttr("json.original", stagingRecord.Json),
			tracing.StringAttr("cod_rms", codRMS),
			tracing.StringAttr("desc_produto", descProduto),
			tracing.StringAttr("procedure", "pkg_integra_produto.prc_integra_hermes"),
			tracing.StringAttr("result.message", result.Message))
	}

	// Always remove from staging after processing (success or failure)
	// This matches the TypeScript behavior: await removeProductService(rms, integrRmsProductInQuery)
	err = uc.repo.RemoveProductIntegrationStagingRecord(stagingRecord.IdIntegrationProdutoStaging)
	if err != nil {
		log.Printf("ERROR: Failed to remove staging record %d: %v", stagingRecord.IdIntegrationProdutoStaging, err)
		// Return the original result but note the removal failure
		return &entities.LogValidate{
			Success: result.Success,
			Message: fmt.Sprintf("%s (Warning: Failed to remove from staging: %v)", result.Message, err),
		}
	}

	log.Printf("Staging record %d removed successfully", stagingRecord.IdIntegrationProdutoStaging)

	return result
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
