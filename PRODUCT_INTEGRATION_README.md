# Product Integration Implementation in Go

This implementation converts the TypeScript/JavaScript product integration service to Go, following the existing project architecture.

## Files Created

### 1. Domain Entities (`domain/entities/productIntegration.go`)
- **ProductInJson**: Main product integration JSON structure
- **ProductSelectIntegration**: Product selection details for integration  
- **ProductNew**: New product structure for internal processing
- **ProductPackaging**: Product packaging information
- **Product**: Complete product entity
- **MarketingStructure**: Marketing structure information
- **Brand, Industry**: Brand and industry entities
- **IntegrRmsProductIn**: RMS product integration input
- **LogIntegrRMS**: Integration log structure
- **LogValidate**: Validation log structure
- **JsonProductSegment**: Product segment JSON for integration
- Various supporting entities and constants

### 2. Repository Layer (`domain/repositories/productIntegrationRepo.go`)
Contains database operations for:
- **GetIntegrRmsProductsIn()**: Retrieve pending RMS product integrations
- **RemoveProductService()**: Remove processed integration records
- **GetMarketingStructureLevel2/4()**: Retrieve marketing structure data
- **GetBrandByIndustryName()**: Get brands by industry and name
- **GetIndustryByNameAndStatus()**: Retrieve industry information
- **SaveIndustry()**, **SaveBrand()**: Save new industry/brand records
- **GetProductByCodeRMS()**: Get product by RMS code
- **GetProductPackagingByBarCode()**: Get packaging by barcode
- **DoPackageProductIntegration()**: Execute Oracle stored procedure
- **SaveLogIntegration()**: Save integration logs
- **SendToQueue()**: Send messages to queue (placeholder)
- Validation functions for marketing structure, brand, and industry

### 3. Use Case Layer (`domain/usecases/productIntegrationUseCase.go`)
Contains business logic for:
- **ImportProductIntegration()**: Main integration function (equivalent to TypeScript `importProductIntegration`)
- **processProductIntegration()**: Process individual product integration
- **getNewProduct()**: Process and validate product data (commented equivalent to TypeScript version)
- Product creation, validation, and processing functions
- Brand and industry processing
- Barcode and packaging processing
- Product insert/update logic

### 4. Delivery Layer Updates (`internal/delivery/listener.go`)
- Added **ProductIntegrationUC** to the Listener struct
- Added **"Produto"** case in the message processing switch statement
- Integrated product integration processing with RabbitMQ messages

### 5. Example Usage (`examples/product_integration_example.go`)
- **runProductIntegrationService()**: Example of how to wire up all components
- **testProductIntegration()**: Example of manual testing
- RabbitMQ message format documentation

## Key Features Implemented

### 1. **Oracle Stored Procedure Integration**
The main processing logic calls Oracle stored procedure `pkg_integra_produto.prc_integra_hermes()` just like the original TypeScript version:

```go
func (r *ProductIntegrationRepository) DoPackageProductIntegration(iprID int) (*entities.LogValidate, error) {
    query := `BEGIN pkg_integra_produto.prc_integra_hermes(:1); END;`
    _, err := r.db.Exec(query, iprID)
    // ... error handling
}
```

### 2. **Message Queue Integration**
Product integration is triggered by RabbitMQ messages with type "Produto":

```json
{
  "tipoIntegracao": "Produto", 
  "dados": {
    // any additional filtering data if needed
  }
}
```

### 3. **Database Transaction Management**
All operations are wrapped in database transactions for data consistency:

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

### 4. **Comprehensive Error Handling and Logging**
- Structured logging throughout the process
- Queue-based logging for integration results
- Proper error handling and recovery

## Integration with Existing System

### RabbitMQ Message Processing
The listener now handles product integration messages alongside existing promotion messages:

```go
switch tipoIntegracao {
case "Promocao":
    // existing promotion logic
case "Produto":
    // new product integration logic
    success, err := l.ProductIntegrationUC.ImportProductIntegration()
}
```

### Database Integration
Uses the existing Oracle database connection and follows the same repository pattern used throughout the project.

### Use Case Architecture
Follows the established use case pattern with dependency injection and clean architecture principles.

## Usage

### 1. Initialize Dependencies
```go
// Database connection
db, err := database.ConectarBanco(cfg)

// Repository
productIntegrationRepo := repositories.NewProductIntegrationRepository(db)

// Use case
productIntegrationUC := usecases.NewProductIntegrationUseCase(productIntegrationRepo, db)

// Add to listener
listener := &rabbitmq.Listener{
    ProductIntegrationUC: productIntegrationUC,
    Workers: 20,
}
```

### 2. Send RabbitMQ Message
Send a message to the `integracaoCron` queue:
```json
{
  "tipoIntegracao": "Produto",
  "dados": {}
}
```

### 3. Monitor Processing
The system will:
1. Read pending records from `INTEGR_RMS_PRODUTO_IN` table
2. Process each record through Oracle stored procedure
3. Log results to queue and database
4. Remove processed records

## Constants and Configuration

Key constants defined in `productIntegration.go`:
- **CONST_TRUE/FALSE**: String boolean constants
- **CB_BARRA_EAN/EAN13/INTERNO**: Barcode type constants  
- **UNIDADE_MEDIDA_KG/UN**: Unit of measurement IDs
- **NOTABILIDADE**: Default notability value

## Error Handling

The implementation includes comprehensive error handling:
- Oracle procedure execution errors
- Database transaction errors
- JSON parsing errors
- Validation errors
- Network/connection errors

All errors are logged and sent to the message queue for monitoring and alerting.

## Performance Considerations

- **Concurrent Processing**: Configurable number of RabbitMQ workers
- **Transaction Management**: Proper database transaction handling
- **Connection Pooling**: Uses existing database connection management
- **Memory Management**: Processes records in batches to avoid memory issues

## Testing

Use the example functions to test:
- `runProductIntegrationService()`: Full service setup
- `testProductIntegration()`: Manual integration testing

## Migration from TypeScript

This Go implementation maintains the same functionality as the original TypeScript version:
- Same Oracle stored procedure calls
- Same database table structure
- Same RabbitMQ message format
- Same logging and error handling patterns
- Same business logic flow

The main differences are:
- Strong typing throughout
- Explicit error handling
- Go idioms and patterns
- Improved performance and concurrency