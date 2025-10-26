package usecases

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/thiagohmm/integracaocron/domain/entities"
	"github.com/thiagohmm/integracaocron/domain/repositories"
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

	result, err := uc.normalizeProducts()
	if err != nil {
		tx.Rollback()
		log.Printf("Erro durante a transação: %v", err)
		return nil, err
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("erro ao fazer commit da transação: %w", err)
	}

	return result, nil
}

// normalizeProducts processes all promotion records and removes duplicates
func (uc *PromotionNormalizationUseCase) normalizeProducts() (*entities.PromotionNormalizationResult, error) {
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

	processedCount := 0
	updatedCount := 0
	totalRemovedDuplicates := 0

	for _, record := range allRecords {
		processError := uc.processRecord(&record, &processedCount, &updatedCount, &totalRemovedDuplicates)
		if processError != nil {
			log.Printf("Error processing record %d: %v", *record.IdIntegracaoPromocao, processError)
			// Continue processing other records even if one fails
		}

		// Log progress every 100 records
		if processedCount%100 == 0 {
			log.Printf("Processados %d registros, %d atualizados", processedCount, updatedCount)
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
	record *entities.PromotionNormalization,
	processedCount *int,
	updatedCount *int,
	totalRemovedDuplicatesGlobal *int,
) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in processRecord for ID %d: %v", *record.IdIntegracaoPromocao, r)
		}
	}()

	*processedCount++

	// Parse the JSON field
	jsonData, err := uc.parseRecordJSON(record)
	if err != nil {
		log.Printf("Erro ao fazer parse do JSON para registro %d: %v", *record.IdIntegracaoPromocao, err)
		return err
	}

	log.Printf("Processing record: %d", *record.IdIntegracaoPromocao)
	log.Printf("Parsed JSON - CodMix: %s, Grupos count: %d", jsonData.CodMix, len(jsonData.Grupos))

	// Normalize groups (remove duplicates)
	hasChanges, totalRemovedDuplicates := uc.repo.NormalizePromotionGroups(jsonData)

	// If changes were made, update the record
	if hasChanges {
		log.Println("Changes detected - updating record")

		updatedJSON, err := json.Marshal(jsonData)
		if err != nil {
			log.Printf("Error marshaling updated JSON: %v", err)
			return err
		}

		log.Printf("updatedJson: %s", string(updatedJSON))

		// Update DataAtualizacao
		now := time.Now()
		record.DataAtualizacao = &now

		// Update the record with the corrected JSON
		err = uc.repo.UpdateRecord(*record, string(updatedJSON))
		if err != nil {
			log.Printf("Error updating record: %v", err)
			return err
		}

		*updatedCount++
		*totalRemovedDuplicatesGlobal += totalRemovedDuplicates

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
	}

	return nil
}

// parseRecordJSON parses the JSON field from a record
func (uc *PromotionNormalizationUseCase) parseRecordJSON(record *entities.PromotionNormalization) (*entities.PromotionJsonData, error) {
	// Handle different JSON representations
	jsonString := record.JSON

	log.Printf("Original record.Json type: %T", jsonString)
	log.Printf("Original record.Json: %s", jsonString)

	// Parse the JSON
	jsonData, err := uc.repo.ParsePromotionJSON(jsonString)
	if err != nil {
		return nil, fmt.Errorf("error parsing JSON: %w", err)
	}

	log.Printf("Final jsonData - CodMix: %s", jsonData.CodMix)

	return jsonData, nil
}

// getIntValue safely gets int value from pointer
func getIntValue(val *int) int {
	if val == nil {
		return 0
	}
	return *val
}
