package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"

	"github.com/thiagohmm/integracaocron/domain/entities"
)

// PromotionNormalizationRepository handles promotion normalization database operations
type PromotionNormalizationRepository struct {
	db *sql.DB
}

// NewPromotionNormalizationRepository creates a new instance of PromotionNormalizationRepository
func NewPromotionNormalizationRepository(db *sql.DB) *PromotionNormalizationRepository {
	return &PromotionNormalizationRepository{
		db: db,
	}
}

// GetAllRecords retrieves all records from the integration promotion table
func (r *PromotionNormalizationRepository) GetAllRecords() ([]entities.PromotionNormalization, error) {
	query := `SELECT ID_INTEGRACAO_PROMOCAO, ID_REVENDEDOR, ID_PROMOCAO, JSON, 
			  DATA_ATUALIZACAO, DATA_RECEBIMENTO, ENVIANDO, TRANSACAO, DATA_INICIO_ENVIO 
			  FROM INTEGRACAO_PROMOCAO 
			  ORDER BY ID_INTEGRACAO_PROMOCAO ASC`

	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("error querying promotion records: %w", err)
	}
	defer rows.Close()

	var results []entities.PromotionNormalization
	for rows.Next() {
		var record entities.PromotionNormalization
		var jsonBytes []byte

		err := rows.Scan(
			&record.IdIntegracaoPromocao,
			&record.IdRevendedor,
			&record.IdPromocao,
			&jsonBytes,
			&record.DataAtualizacao,
			&record.DataRecebimento,
			&record.Enviando,
			&record.Transacao,
			&record.DataInicioEnvio,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning promotion record: %w", err)
		}

		// Convert JSON bytes to string
		record.JSON = string(jsonBytes)
		results = append(results, record)
	}

	return results, nil
}

// UpdateRecord updates a promotion record with normalized JSON
func (r *PromotionNormalizationRepository) UpdateRecord(record entities.PromotionNormalization, updatedJSON string) error {
	query := `UPDATE INTEGRACAO_PROMOCAO 
			  SET JSON = :1, DATA_ATUALIZACAO = :2 
			  WHERE ID_INTEGRACAO_PROMOCAO = :3 
			    AND ID_REVENDEDOR = :4 
			    AND ID_PROMOCAO = :5`

	_, err := r.db.Exec(query,
		updatedJSON,
		record.DataAtualizacao,
		record.IdIntegracaoPromocao,
		record.IdRevendedor,
		record.IdPromocao,
	)
	if err != nil {
		return fmt.Errorf("error updating promotion record: %w", err)
	}

	return nil
}

// SendToQueue sends a message to the queue (placeholder implementation)
func (r *PromotionNormalizationRepository) SendToQueue(message entities.QueueMessage) error {
	// This would integrate with RabbitMQ or other message queue
	// For now, we'll just log the message
	messageJSON, _ := json.Marshal(message)
	log.Printf("Sending to queue: %s", string(messageJSON))
	return nil
}

// ParsePromotionJSON parses and validates promotion JSON data
func (r *PromotionNormalizationRepository) ParsePromotionJSON(jsonStr string) (*entities.PromotionJsonData, error) {
	var promotionData entities.PromotionJsonData

	if err := json.Unmarshal([]byte(jsonStr), &promotionData); err != nil {
		return nil, fmt.Errorf("error parsing promotion JSON: %w", err)
	}

	return &promotionData, nil
}

// NormalizePromotionGroups removes duplicate items from promotion groups based on codBarra
func (r *PromotionNormalizationRepository) NormalizePromotionGroups(data *entities.PromotionJsonData) (bool, int) {
	hasChanges := false
	totalRemovedDuplicates := 0

	if len(data.Grupos) == 0 {
		log.Println("No grupos array found or grupos is empty")
		return hasChanges, totalRemovedDuplicates
	}

	log.Printf("Found grupos array with %d groups", len(data.Grupos))

	for i, grupo := range data.Grupos {
		log.Printf("Processing group %d: %s", i+1, grupo.Desc)

		if len(grupo.Items) == 0 {
			log.Printf("Group %d - no items array or empty items", i+1)
			continue
		}

		log.Printf("Group %d has %d items", i+1, len(grupo.Items))

		uniqueItems := []entities.PromotionGroupItem{}
		seen := make(map[string]bool)

		// Remove duplicates from items array based on codBarra
		for _, item := range grupo.Items {
			if item.CodBarra != "" && !seen[item.CodBarra] {
				seen[item.CodBarra] = true
				uniqueItems = append(uniqueItems, item)
			}
		}

		log.Printf("Group %d - Original items: %d, Unique items: %d", i+1, len(grupo.Items), len(uniqueItems))

		// If duplicates were found and removed, update the group
		if len(uniqueItems) != len(grupo.Items) {
			log.Printf("Updating group %d - removing %d duplicates", i+1, len(grupo.Items)-len(uniqueItems))
			data.Grupos[i].Items = uniqueItems
			// Update qtdeItem to reflect the new count after removing duplicates
			data.Grupos[i].QtdeItem = len(uniqueItems)
			hasChanges = true
			totalRemovedDuplicates += len(grupo.Items) - len(uniqueItems)
		} else {
			log.Printf("Group %d - no duplicates found", i+1)
		}
	}

	return hasChanges, totalRemovedDuplicates
}

// CreateLogMessage creates a log message for queue
func (r *PromotionNormalizationRepository) CreateLogMessage(transacao, tabela, descricao string, jsonData interface{}) entities.QueueMessage {
	return entities.QueueMessage{
		Tabela: "LogIntegrRMS",
		Fields: []string{"TRANSACAO", "TABELA", "DATARECEBIMENTO", "DATAPROCESSAMENTO", "STATUSPROCESSAMENTO", "JSON", "DESCRICAOERRO"},
		Values: []interface{}{
			transacao,
			tabela,
			"SYSDATE", // Oracle function for current date
			"SYSDATE", // Oracle function for current date
			1,         // Status: 1 for success, 0 for error
			r.marshalToJSON(jsonData),
			descricao,
		},
	}
}

// CreateErrorLogMessage creates an error log message for queue
func (r *PromotionNormalizationRepository) CreateErrorLogMessage(transacao, tabela, descricao string, jsonData interface{}) entities.QueueMessage {
	return entities.QueueMessage{
		Tabela: "LogIntegrRMS",
		Fields: []string{"TRANSACAO", "TABELA", "DATARECEBIMENTO", "DATAPROCESSAMENTO", "STATUSPROCESSAMENTO", "JSON", "DESCRICAOERRO"},
		Values: []interface{}{
			transacao,
			tabela,
			"SYSDATE", // Oracle function for current date
			"SYSDATE", // Oracle function for current date
			0,         // Status: 0 for error
			r.marshalToJSON(jsonData),
			descricao,
		},
	}
}

// marshalToJSON safely marshals data to JSON string
func (r *PromotionNormalizationRepository) marshalToJSON(data interface{}) string {
	jsonBytes, err := json.Marshal(data)
	if err != nil {
		log.Printf("Error marshaling data to JSON: %v", err)
		return "{}"
	}
	return string(jsonBytes)
}
