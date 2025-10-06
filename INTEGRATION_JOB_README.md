# Integration Job Implementation - Go

This document explains how to use the Go equivalent of the `productNetworkMain` TypeScript function that you wanted to call at the end of each use case.

## Overview

I've successfully transformed your complex TypeScript integration job into a comprehensive Go implementation with the following components:

### 1. **IntegrationJobUseCase** (`domain/usecases/integrationJobUseCase.go`)

This is the main use case that implements the equivalent of your `productNetworkMain` function. It includes:

- **ProductNetworkMain()** - The main function that orchestrates all integration jobs with transaction management
- **IntegrationJob()** - Handles cleanup and expiry operations
- **ReplicateNetworkProductsJob()** - Replicates products across networks
- **MoveDataJob()** - Moves data between staging tables
- **UpdateExpirationSlaRequestsJob()** - Updates expired SLA requests

### 2. **Repository Implementations**

- **ParameterRepository** - Manages system parameters
- **IntegrationRepository** - Handles integration cleanup and data movement operations
- **NetworkRepository** - Manages network and product replication operations

### 3. **Integration with Existing Use Cases**

The `PromotionUseCase` has been updated to automatically call the integration job at the end of processing:

```go
// In ProcessIntegrationPromotions method
if uc.integrationJobUC != nil {
    log.Println("Chamando job de integração no final do processamento de promoção...")
    dataCorte := time.Now()
    if err := uc.integrationJobUC.ProductNetworkMain(dataCorte); err != nil {
        log.Printf("Erro ao executar job de integração: %v", err)
        // Error is logged but doesn't fail the main promotion processing
    }
}
```

## Key Features Implemented

### ✅ **Transaction Management**
- Database transactions with proper rollback on errors
- Similar to the TypeScript `transaction.commit()` and `transaction.rollback()`

### ✅ **Date Formatting for Oracle** 
- `FormatDateForOracle()` function that converts Go time.Time to Oracle timestamp format
- Equivalent to the TypeScript `formatDateForOracle()` function

### ✅ **Parameter Management**
- Retrieval and updating of system parameters
- Equivalent to `getValueParameterRemoveTransactionJob()`, etc.

### ✅ **Integration Operations**
- Transaction removal with configurable expiry
- Data movement between staging tables
- Network product replication
- SLA expiration updates

### ✅ **Error Handling**
- Comprehensive error handling with logging
- Graceful degradation (integration job errors don't fail main processing)

## Usage Examples

### Basic Usage (Single Promotion)
```go
// Create repositories
promotionRepo := repositories.NewPromotionRepository(db)
parameterRepo := repositories.NewParameterRepository(db)
integrationRepo := repositories.NewIntegrationRepository(db)
networkRepo := repositories.NewNetworkRepository(db)

// Create integration job use case
integrationJobUC := usecases.NewIntegrationJobUseCase(parameterRepo, integrationRepo, networkRepo, db)

// Create promotion use case with integration job
promotionUC := usecases.NewPromotionUseCase(promotionRepo, rabbitmqURL, integrationJobUC)

// Process promotion - integration job will be called automatically at the end
promotion := entities.Promotion{IPMD_ID: 123, Json: `{"test": "data"}`, DATARECEBIMENTO: "2025-10-06 12:00:00"}
err := promotionUC.ProcessIntegrationPromotions(promotion)
```

### Direct Integration Job Usage
```go
// Call the integration job directly
integrationJobUC := usecases.NewIntegrationJobUseCase(parameterRepo, integrationRepo, networkRepo, db)
dataCorte := time.Now()
err := integrationJobUC.ProductNetworkMain(dataCorte)
```

## Database Tables Expected

The implementation assumes these Oracle tables exist (adjust table names as needed):

- `PARAMETROS` - System parameters
- `INTEGR_RMS_PROMOCAO_IN` - Promotion integration staging
- `INTEGR_COMBO` - Combo integration
- `INTEGR_EMBALAGEM` - Packaging integration  
- `INTEGR_ESTRUTURA_MERCADOLOGICA` - Marketing structure integration
- `INTEGR_PRODUTO` - Product integration
- `INTEGR_PROMOCAO` - Promotion integration
- `REDES` - Networks
- `REVENDEDOR_REDE` - Dealer networks
- `PRODUTOS_REPLICADOS` - Replicated products

## Configuration Parameters

The system expects these parameters in the `PARAMETROS` table:

- `REMOVER_TRANSACAO_MINUTOS` - Minutes to subtract for transaction cleanup
- `EXPURGO_INTEGRACAO_DIAS` - Days for integration expiry
- `Parametro_ExpurgoIntegracaoUltimaExecucao` - Last expiry execution timestamp
- `RemoverTransacaoUltimaExecucao` - Last transaction cleanup timestamp

## How to Extend to Other Use Cases

To add the integration job to other use cases (like products, combos, etc.):

1. **Add IntegrationJobUseCase to your use case struct:**
```go
type ProductUseCase struct {
    productRepo      entities.ProductRepository
    integrationJobUC *IntegrationJobUseCase
}
```

2. **Call it at the end of processing:**
```go
func (uc *ProductUseCase) ProcessProduct(product entities.Product) error {
    // Your product processing logic here
    
    // Call integration job at the end
    if uc.integrationJobUC != nil {
        dataCorte := time.Now()
        if err := uc.integrationJobUC.ProductNetworkMain(dataCorte); err != nil {
            log.Printf("Erro ao executar job de integração: %v", err)
        }
    }
    
    return nil
}
```

## Benefits of This Implementation

1. **Type Safety** - Go's type system prevents many runtime errors
2. **Performance** - Compiled Go code runs faster than interpreted TypeScript
3. **Concurrency** - Easy to add Go routines for parallel processing
4. **Error Handling** - Explicit error handling makes debugging easier
5. **Transaction Safety** - Proper database transaction management
6. **Modularity** - Clean separation of concerns with repositories and use cases

The Go implementation maintains all the functionality of your original TypeScript code while providing better performance, type safety, and maintainability.