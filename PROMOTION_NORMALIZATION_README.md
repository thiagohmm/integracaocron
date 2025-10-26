# Promotion Normalization Implementation in Go

This implementation converts the TypeScript/JavaScript promotion normalization service to Go, following the existing project architecture.

## Overview

The Promotion Normalization service processes promotion data stored in the `INTEGRACAO_PROMOCAO` table and removes duplicate items from promotion groups based on barcode (`codBarra`). This ensures data quality and prevents duplicate products in promotional campaigns.

## Files Created

### 1. Domain Entities (`domain/entities/promotionNormalization.go`)
- **PromotionNormalization**: Main database record structure
- **PromotionJsonData**: Structure of the JSON field containing promotion details
- **PromotionGroup**: Group within a promotion with items
- **PromotionGroupItem**: Individual item with barcode, description, price, and quantity
- **PromotionNormalizationResult**: Result structure for normalization process
- **PromotionNormalizationLog**: Log information structure
- Constants for messages and promotion types

### 2. Repository Layer (`domain/repositories/promotionNormalizationRepo.go`)
Contains database operations for:
- **GetAllRecords()**: Retrieve all promotion records from INTEGRACAO_PROMOCAO
- **UpdateRecord()**: Update record with normalized JSON
- **ParsePromotionJSON()**: Parse and validate JSON data
- **NormalizePromotionGroups()**: Remove duplicate items from groups
- **CreateLogMessage()**: Create success log messages
- **CreateErrorLogMessage()**: Create error log messages
- **SendToQueue()**: Send messages to queue for monitoring

### 3. Use Case Layer (`domain/usecases/promotionNormalizationUseCase.go`)
Contains business logic for:
- **NormalizePromotions()**: Main normalization function with transaction management
- **normalizeProducts()**: Process all promotion records
- **processRecord()**: Process individual promotion record
- **parseRecordJSON()**: Parse JSON from database record
- Helper functions for safe value extraction

### 4. Delivery Layer Updates (`internal/delivery/listener.go`)
- Added **PromotionNormalizationUC** to the Listener struct
- Added **"PromocaoNormalizacao"** case in message processing
- Integrated normalization with RabbitMQ message handling

### 5. Example Usage (`examples/promotion_normalization_example.go`)
- **runPromotionNormalizationService()**: Service setup example
- **testPromotionNormalization()**: Manual testing example
- Detailed documentation of message format and JSON structure

## How It Works

### 1. **Data Structure**

Promotion data in the database has this JSON structure:

```json
{
  "codMix": "12345",
  "grupos": [
    {
      "desc": "Group 1",
      "qtdeItem": 5,
      "items": [
        {"codBarra": "123", "desc": "Product 1", "preco": 10.0, "qtde": 1},
        {"codBarra": "123", "desc": "Product 1", "preco": 10.0, "qtde": 1},
        {"codBarra": "456", "desc": "Product 2", "preco": 20.0, "qtde": 2}
      ]
    }
  ]
}
```

### 2. **Normalization Process**

The service:
1. Reads all records from `INTEGRACAO_PROMOCAO` table
2. For each record, parses the JSON field
3. For each group (`grupos`) in the promotion:
   - Identifies duplicate items based on `codBarra` (barcode)
   - Keeps only unique items
   - Updates `qtdeItem` to reflect the actual count
4. Updates the record with normalized JSON
5. Logs changes to message queue

### 3. **Duplicate Detection**

Duplicates are identified by matching `codBarra` values. The first occurrence is kept, subsequent duplicates are removed:

```go
uniqueItems := []entities.PromotionGroupItem{}
seen := make(map[string]bool)

for _, item := range grupo.Items {
    if item.CodBarra != "" && !seen[item.CodBarra] {
        seen[item.CodBarra] = true
        uniqueItems = append(uniqueItems, item)
    }
}
```

### 4. **Transaction Management**

All operations are wrapped in database transactions:

```go
tx, err := uc.db.Begin()
defer func() {
    if p := recover(); p != nil {
        tx.Rollback()
        panic(p)
    }
}()
// ... processing
tx.Commit()
```

## RabbitMQ Integration

### Message Format

Send messages to the `integracaoCron` queue:

```json
{
  "tipoIntegracao": "PromocaoNormalizacao",
  "dados": {}
}
```

The `dados` field can contain optional filtering parameters if needed in the future.

### Message Processing Flow

1. **Listener** receives message from RabbitMQ
2. **Routes** to PromotionNormalizationUC based on `tipoIntegracao`
3. **Processes** all records in database
4. **Logs** results to queue
5. **Returns** success/failure status

## Database Integration

### Table Structure

Works with the `INTEGRACAO_PROMOCAO` table:

```sql
CREATE TABLE INTEGRACAO_PROMOCAO (
    ID_INTEGRACAO_PROMOCAO NUMBER PRIMARY KEY,
    ID_REVENDEDOR NUMBER,
    ID_PROMOCAO NUMBER,
    JSON CLOB,
    DATA_ATUALIZACAO TIMESTAMP,
    DATA_RECEBIMENTO TIMESTAMP,
    ENVIANDO VARCHAR2(1),
    TRANSACAO VARCHAR2(20),
    DATA_INICIO_ENVIO TIMESTAMP
)
```

### Update Operations

The service updates records with:
- Normalized JSON (duplicates removed)
- Current timestamp in `DATA_ATUALIZACAO`

## Logging and Monitoring

### Success Logs

For each updated record:

```go
logData := entities.PromotionNormalizationLog{
    IdIntegracaoPromocao: record.IdIntegracaoPromocao,
    IdPromocao:          record.IdPromocao,
    IdRevendedor:        record.IdRevendedor,
    CodMix:              jsonData.CodMix,
    RemovedDuplicates:   totalRemovedDuplicates,
}
```

Sent to queue with:
- Transaction type: "UPDATE"
- Table: "INTEGRACAOPROMOCAOSTAGING"
- Status: 1 (success)
- Description: "Itens duplicados removidos dos grupos. Total removidos: X"

### Error Logs

For processing errors:
- Transaction type: "UPDATE"
- Table: "INTEGRACAOPROMOCAOSTAGING"
- Status: 0 (error)
- Description: Error message with stack trace

### Progress Logging

Console logs every 100 records:
```
Processados 100 registros, 45 atualizados
Processados 200 registros, 89 atualizados
```

## Usage Examples

### 1. Initialize Service

```go
db, err := database.ConectarBanco(cfg)
promotionNormalizationRepo := repositories.NewPromotionNormalizationRepository(db)
promotionNormalizationUC := usecases.NewPromotionNormalizationUseCase(promotionNormalizationRepo, db)

listener := &rabbitmq.Listener{
    PromotionNormalizationUC: promotionNormalizationUC,
    Workers: 20,
}
```

### 2. Trigger via RabbitMQ

```bash
# Publish message to queue
rabbitmqadmin publish routing_key=integracaoCron \
  payload='{"tipoIntegracao":"PromocaoNormalizacao","dados":{}}'
```

### 3. Manual Execution

```go
result, err := promotionNormalizationUC.NormalizePromotions()
if err != nil {
    log.Printf("Error: %v", err)
    return
}

log.Printf("Processed: %d, Updated: %d, Duplicates: %d",
    result.ProcessedCount,
    result.UpdatedCount,
    result.TotalRemovedDuplicates)
```

## Performance Considerations

### Batch Processing
- Processes all records in memory
- For very large datasets, consider implementing pagination

### Concurrency
- Currently processes records sequentially
- Transaction ensures data consistency
- Workers handle concurrent RabbitMQ messages

### Memory Usage
- JSON parsing done per record
- Records processed one at a time
- Go's garbage collection handles cleanup

## Error Handling

### Panic Recovery
All processing functions include panic recovery:
```go
defer func() {
    if r := recover(); r != nil {
        log.Printf("Panic recovered: %v", r)
        // Send error log to queue
    }
}()
```

### Partial Failures
- If one record fails, processing continues
- Each error is logged separately
- Final result includes counts of processed/updated records

### Transaction Rollback
- Any panic triggers transaction rollback
- Database remains consistent
- Error logged to queue for investigation

## Monitoring and Metrics

### Key Metrics
- **ProcessedCount**: Total records examined
- **UpdatedCount**: Records actually modified
- **TotalRemovedDuplicates**: Items removed across all groups

### Health Checks
Monitor logs for:
- Processing completion messages
- Error rates
- Update percentages
- Queue message delivery

## Migration from TypeScript

### Equivalents

| TypeScript | Go |
|-----------|-----|
| `normalizeProducts()` | `NormalizePromotions()` |
| `JSON.parse()` | `json.Unmarshal()` |
| `Buffer.isBuffer()` | Handled in SQL driver |
| `sendToQueue()` | `repo.SendToQueue()` |
| `db_connect.transaction()` | `db.Begin()` / `tx.Commit()` |

### Differences
- **Type Safety**: Go enforces types at compile time
- **Error Handling**: Explicit error checking vs try/catch
- **Null Handling**: Pointers for nullable values
- **Concurrency**: Goroutines instead of async/await
- **Performance**: Compiled binary vs interpreted JS

## Testing

### Unit Tests
Test individual components:
```go
func TestNormalizePromotionGroups(t *testing.T) {
    repo := repositories.NewPromotionNormalizationRepository(db)
    data := &entities.PromotionJsonData{
        // test data
    }
    hasChanges, removed := repo.NormalizePromotionGroups(data)
    // assertions
}
```

### Integration Tests
Test full workflow with test database:
```go
func TestNormalizePromotions(t *testing.T) {
    // Setup test database
    // Insert test records
    // Run normalization
    // Verify results
}
```

## Troubleshooting

### Common Issues

**1. JSON Parse Errors**
- Check CLOB encoding in Oracle
- Verify JSON structure matches entities
- Log original JSON for debugging

**2. No Records Updated**
- Check if records have actual duplicates
- Verify database connection
- Check transaction commit

**3. Performance Issues**
- Monitor record count
- Consider pagination for large datasets
- Check database query performance

## Future Enhancements

Potential improvements:
1. **Filtering**: Add parameters to process specific promotions
2. **Pagination**: Process records in batches
3. **Parallel Processing**: Use goroutines for multiple records
4. **Metrics**: Add Prometheus metrics
5. **Configuration**: Make table names configurable
6. **Dry Run**: Preview changes without updating

## Dependencies

- **database/sql**: Database operations
- **encoding/json**: JSON parsing
- **github.com/streadway/amqp**: RabbitMQ client
- Oracle database driver (godror or go-ora)

## Conclusion

This Go implementation provides:
- ✅ Full feature parity with TypeScript version
- ✅ Strong typing and compile-time safety
- ✅ Robust error handling and recovery
- ✅ Comprehensive logging and monitoring
- ✅ Transaction management for data consistency
- ✅ Clean architecture with separation of concerns
- ✅ Easy integration with existing services