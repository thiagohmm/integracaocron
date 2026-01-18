package usecases

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/thiagohmm/integracaocron/domain/entities"
	"github.com/thiagohmm/integracaocron/domain/repositories"
	"github.com/thiagohmm/integracaocron/pkg/tracing"
)

// PromotionNormalizationUseCase handles promotion normalization business logic
type PromotionNormalizationUseCase struct {
	repo *repositories.PromotionNormalizationRepository
	db   *sql.DB
}

// NewPromotionNormalizationUseCase creates a new instance of PromotionNormalizationUseCase
func NewPromotionNormalizationUseCase(repo *repositories.PromotionNormalizationRepository, db *sql.DB) *PromotionNormalizationUseCase {
	return &PromotionNormalizationUseCase{
		repo: repo,
		db:   db,
	}
}

// NormalizePromotions is the main function that normalizes promotion data
func (uc *PromotionNormalizationUseCase) NormalizePromotions() (*entities.PromotionNormalizationResult, error) {
	ctx := context.Background()
	ctx, span := tracing.StartSpan(ctx, "NormalizePromotions")
	defer span.End()
	
	log.Println(entities.MSG_START_IMPORT_PROMOTION_RMS)
	defer log.Println(entities.MSG_END_IMPORT_PROMOTION_RMS)

	// Begin transaction
	tx, err := uc.db.Begin()
	if err != nil {
		return nil, fmt.Errorf("erro ao iniciar transação: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	result, err := uc.normalizeProducts(ctx)
	if err != nil {
		tx.Rollback()
		log.Printf("Erro durante a transação: %v", err)
		tracing.RecordError(ctx, err)
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		tracing.RecordError(ctx, err)
		return nil, fmt.Errorf("erro ao fazer commit da transação: %w", err)
	}

	// ✅ Adicionar estatísticas finais no Jaeger
	if result != nil {
		tracing.AddIntAttribute(ctx, "normalization.processed_count", result.ProcessedCount)
		tracing.AddIntAttribute(ctx, "normalization.updated_count", result.UpdatedCount)
		tracing.AddIntAttribute(ctx, "normalization.removed_duplicates", result.TotalRemovedDuplicates)
		tracing.AddBoolAttribute(ctx, "normalization.success", result.Success)
		tracing.AddStringAttribute(ctx, "normalization.message", result.Message)
	}

	return result, nil
}

// normalizeProducts processes all promotion records and removes duplicates
func (uc *PromotionNormalizationUseCase) normalizeProducts(ctx context.Context) (*entities.PromotionNormalizationResult, error) {
	result := &entities.PromotionNormalizationResult{
		Success: true,
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in normalizeProducts: %v", r)
			result.Success = false
			result.Message = fmt.Sprintf("Panic: %v", r)

			// Send error log to queue
			errorMsg := uc.repo.CreateErrorLogMessage(
				"UPDATE",
				"INTEGRACAOPROMOCAOSTAGING",
				fmt.Sprintf("Panic during normalization: %v", r),
				map[string]interface{}{"error": fmt.Sprintf("%v", r)},
			)
			uc.repo.SendToQueue(errorMsg)
		}
	}()

	// Get all records from the staging table
	allRecords, err := uc.repo.GetAllRecords()
	if err != nil {
		errMsg := fmt.Sprintf("Erro ao obter registros: %v", err)
		log.Println(errMsg)

		errorLog := uc.repo.CreateErrorLogMessage(
			"UPDATE",
			"INTEGRACAOPROMOCAOSTAGING",
			errMsg,
			map[string]interface{}{"error": err.Error()},
		)
		uc.repo.SendToQueue(errorLog)

		return nil, fmt.Errorf("erro ao obter registros: %w", err)
	}

	log.Printf("Total records to process: %d", len(allRecords))
	tracing.AddIntAttribute(ctx, "normalization.total_records", len(allRecords))

	processedCount := 0
	updatedCount := 0
	totalRemovedDuplicates := 0

	for _, record := range allRecords {
		processError := uc.processRecord(ctx, &record, &processedCount, &updatedCount, &totalRemovedDuplicates)
		if processError != nil {
			log.Printf("Error processing record %d: %v", *record.IdIntegracaoPromocao, processError)
			// Continue processing other records even if one fails
		}

		// Log progress every 100 records
		if processedCount%100 == 0 {
			log.Printf("Processados %d registros, %d atualizados", processedCount, updatedCount)
			tracing.AddIntAttribute(ctx, "normalization.progress.processed", processedCount)
			tracing.AddIntAttribute(ctx, "normalization.progress.updated", updatedCount)
		}
	}

	log.Printf("Processamento concluído. Total processados: %d, Total atualizados: %d", processedCount, updatedCount)

	result.ProcessedCount = processedCount
	result.UpdatedCount = updatedCount
	result.TotalRemovedDuplicates = totalRemovedDuplicates
	result.Message = fmt.Sprintf("Processamento concluído. Total processados: %d, Total atualizados: %d", processedCount, updatedCount)

	return result, nil
}

// processRecord processes a single promotion record
func (uc *PromotionNormalizationUseCase) processRecord(
	ctx context.Context,
	record *entities.PromotionNormalization,
	processedCount *int,
	updatedCount *int,
	totalRemovedDuplicatesGlobal *int,
) error {
	// ✅ Criar span para cada registro processado
	recordCtx, span := tracing.StartSpan(ctx, "ProcessPromotionRecord")
	defer span.End()
	
	// ✅ Adicionar IDs para pesquisa no Jaeger
	if record.IdIntegracaoPromocao != nil {
		tracing.AddPromotionID(recordCtx, *record.IdIntegracaoPromocao)
	}
	if record.IdPromocao != nil {
		tracing.AddIntAttribute(recordCtx, "promotion.id_promocao", *record.IdPromocao)
	}
	if record.IdRevendedor != nil {
		tracing.AddIntAttribute(recordCtx, "promotion.id_revendedor", *record.IdRevendedor)
	}
	
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in processRecord for ID %d: %v", *record.IdIntegracaoPromocao, r)
			tracing.AddEvent(recordCtx, "panic.recovered", tracing.StringAttr("panic_value", fmt.Sprintf("%v", r)))
		}
	}()

	*processedCount++

	// ✅ Adicionar JSON original no Jaeger ANTES do parse
	tracing.AddStringAttribute(recordCtx, "promotion.json.original", record.JSON)
	tracing.AddEvent(recordCtx, "json.original.logged", tracing.StringAttr("json_length", fmt.Sprintf("%d", len(record.JSON))))

	// Parse the JSON field
	jsonData, err := uc.parseRecordJSON(recordCtx, record)
	if err != nil {
		log.Printf("Erro ao fazer parse do JSON para registro %d: %v", *record.IdIntegracaoPromocao, err)
		tracing.RecordError(recordCtx, err)
		return err
	}

	log.Printf("Processing record: %d", *record.IdIntegracaoPromocao)
	log.Printf("Parsed JSON - CodMix: %s, Grupos count: %d", jsonData.CodMix, len(jsonData.Grupos))
	
	// ✅ Adicionar informações do JSON parseado no Jaeger
	tracing.AddStringAttribute(recordCtx, "promotion.cod_mix", jsonData.CodMix)
	tracing.AddIntAttribute(recordCtx, "promotion.grupos.count", len(jsonData.Grupos))

	// Normalize groups (remove duplicates)
	hasChanges, totalRemovedDuplicates := uc.repo.NormalizePromotionGroups(jsonData)
	
	// ✅ Adicionar informações sobre normalização no Jaeger
	tracing.AddBoolAttribute(recordCtx, "promotion.normalization.has_changes", hasChanges)
	tracing.AddIntAttribute(recordCtx, "promotion.normalization.removed_duplicates", totalRemovedDuplicates)
	
	// Adicionar detalhes de cada grupo
	for i, grupo := range jsonData.Grupos {
		tracing.AddIntAttribute(recordCtx, fmt.Sprintf("promotion.grupo.%d.items_count", i+1), len(grupo.Items))
		tracing.AddStringAttribute(recordCtx, fmt.Sprintf("promotion.grupo.%d.desc", i+1), grupo.Desc)
	}

	// If changes were made, update the record
	if hasChanges {
		log.Println("Changes detected - updating record")

		updatedJSON, err := json.Marshal(jsonData)
		if err != nil {
			log.Printf("Error marshaling updated JSON: %v", err)
			tracing.RecordError(recordCtx, err)
			return err
		}

		log.Printf("updatedJson: %s", string(updatedJSON))
		
		// ✅ Adicionar JSON atualizado no Jaeger
		tracing.AddStringAttribute(recordCtx, "promotion.json.updated", string(updatedJSON))
		tracing.AddEvent(recordCtx, "json.updated.logged")

		// Update DataAtualizacao
		now := time.Now()
		record.DataAtualizacao = &now

		// Update the record with the corrected JSON
		err = uc.repo.UpdateRecord(*record, string(updatedJSON))
		if err != nil {
			log.Printf("Error updating record: %v", err)
			tracing.RecordError(recordCtx, err)
			return err
		}

		*updatedCount++
		*totalRemovedDuplicatesGlobal += totalRemovedDuplicates
		
		// ✅ Marcar como atualizado no Jaeger
		tracing.AddEvent(recordCtx, "record.updated", 
			tracing.IntAttr("removed_duplicates", totalRemovedDuplicates))
		tracing.SetStatus(recordCtx, 1, "Record updated successfully")

		// Log the update
		logData := entities.PromotionNormalizationLog{
			IdIntegracaoPromocao: getIntValue(record.IdIntegracaoPromocao),
			IdPromocao:           getIntValue(record.IdPromocao),
			IdRevendedor:         getIntValue(record.IdRevendedor),
			CodMix:               jsonData.CodMix,
			RemovedDuplicates:    totalRemovedDuplicates,
		}

		logSucesso := uc.repo.CreateLogMessage(
			"UPDATE",
			"INTEGRACAOPROMOCAOSTAGING",
			fmt.Sprintf("Itens duplicados removidos dos grupos. Total removidos: %d", totalRemovedDuplicates),
			logData,
		)
		uc.repo.SendToQueue(logSucesso)
	} else {
		log.Println("No changes detected - record not updated")
		tracing.AddEvent(recordCtx, "record.no_changes")
		tracing.SetStatus(recordCtx, 1, "No changes needed")
	}

	return nil
}

// parseRecordJSON parses the JSON field from a record
func (uc *PromotionNormalizationUseCase) parseRecordJSON(ctx context.Context, record *entities.PromotionNormalization) (*entities.PromotionJsonData, error) {
	// Handle different JSON representations
	jsonString := record.JSON

	log.Printf("Original record.Json type: %T", jsonString)
	log.Printf("Original record.Json: %s", jsonString)
	
	// ✅ O JSON original já foi adicionado no span pai, mas podemos adicionar evento aqui também
	tracing.AddEvent(ctx, "json.parse.start", tracing.StringAttr("json_length", fmt.Sprintf("%d", len(jsonString))))

	// Parse the JSON
	jsonData, err := uc.repo.ParsePromotionJSON(jsonString)
	if err != nil {
		tracing.RecordError(ctx, err)
		tracing.AddEvent(ctx, "json.parse.failed")
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	log.Printf("Final jsonData - CodMix: %s", jsonData.CodMix)
	tracing.AddEvent(ctx, "json.parse.success", tracing.StringAttr("cod_mix", jsonData.CodMix))

	return jsonData, nil
}

// getIntValue safely gets int value from pointer
func getIntValue(val *int) int {
	if val == nil {
		return 0
	}
	return *val
}
