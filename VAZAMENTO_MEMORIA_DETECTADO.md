# 🔴 VAZAMENTOS DE MEMÓRIA IDENTIFICADOS

## ❌ Problemas Críticos Encontrados

### 1. **Transação NUNCA é fechada (GRAVE)**

**Arquivo:** `domain/usecases/productIntegrationUseCase.go` linha 71

```go
// ❌ PROBLEMA: Transaction NÃO é usada mas NUNCA é fechada
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

// ... processo produtos ...

// ❌ COMMIT SEM VERIFICAÇÃO DE ERRO ANTERIOR
if err := tx.Commit(); err != nil {
    return false, fmt.Errorf("error committing transaction: %w", err)
}
```

**Problema:**
- Se houver erro antes do commit, a transação NUNCA é fechada
- Cada execução deixa uma conexão aberta no pool
- Causa esgotamento de conexões do banco

---

### 2. **Array `success` cresce infinitamente**

**Arquivo:** `domain/usecases/productIntegrationUseCase.go`

```go
var success []bool  // ❌ Cresce sem limite

for i, stagingRecord := range productsStagingRecords {
    // ...
    success = append(success, true/false)  // ❌ Append infinito
}
```

**Problema:**
- Com 10.000 produtos = 10.000 booleans na memória
- Desnecessário, só precisa de contadores

---

### 3. **Strings e JSON acumulam na memória**

**Arquivo:** `domain/usecases/productIntegrationUseCase.go`

```go
logErro := entities.QueueMessage{
    // ...
    Values: []interface{}{
        "STAGING",
        "PRODUTOS",
        stagingRecord.DataAtualizacao,
        time.Now(),
        uc.getStatusFromResult(result),
        stagingRecord.Json,  // ❌ JSON completo copiado para memória
        uc.getMessageFromResult(result),
    },
}
```

**Problema:**
- Copia JSON completo de cada produto para array
- Com produtos grandes, consome muita RAM

---

### 4. **Loop aninhado N x M x P (LENTIDÃO + MEMÓRIA)**

**Arquivo:** `domain/usecases/integrationJobUseCase.go`

```go
for _, net := range networks {              // N redes
    for _, loja := range lojas {            // M lojas por rede
        for _, product := range products {  // P produtos por loja
            // ❌ N * M * P iterações
        }
    }
}
```

**Problema:**
- 10 redes × 100 lojas × 1000 produtos = **1.000.000 iterações**
- Cada iteração cria objetos temporários
- Garbage Collector sobrecarregado

---

### 5. **Contexto sem timeout em algumas queries**

**Arquivo:** `domain/repositories/networkRepo.go`

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

**Problema Parcial:**
- 30 segundos pode ser muito para queries simples
- Queries longas bloqueiam memória

---

## ✅ CORREÇÕES NECESSÁRIAS

### Correção 1: Garantir fechamento de transação

```go
tx, err := uc.db.Begin()
if err != nil {
    return false, fmt.Errorf("error starting transaction: %w", err)
}

// ✅ SEMPRE fecha a transação (commit ou rollback)
committed := false
defer func() {
    if !committed {
        if err := tx.Rollback(); err != nil {
            log.Printf("Error rolling back transaction: %v", err)
        }
    }
}()

// ... processar ...

if err := tx.Commit(); err != nil {
    return false, fmt.Errorf("error committing transaction: %w", err)
}
committed = true  // ✅ Marca como committed
```

---

### Correção 2: Usar contadores ao invés de array

```go
// ❌ ANTES
var success []bool
success = append(success, true)

// ✅ DEPOIS
var successCount, failureCount int
if result.Success {
    successCount++
} else {
    failureCount++
}
```

---

### Correção 3: Processar em batches menores

```go
// ✅ Processar 100 produtos por vez
batchSize := 100
for i := 0; i < len(productsStagingRecords); i += batchSize {
    end := i + batchSize
    if end > len(productsStagingRecords) {
        end = len(productsStagingRecords)
    }
    batch := productsStagingRecords[i:end]
    
    // Processar batch
    // Liberar memória após cada batch
    runtime.GC()  // Força coleta de lixo
}
```

---

### Correção 4: Reduzir timeout de queries

```go
// ✅ Timeout apropriado para tipo de query
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)  // Queries rápidas
defer cancel()
```

---

### Correção 5: Limitar tamanho de arrays retornados

```go
// ✅ Adicionar LIMIT nas queries
query := `SELECT ... FROM ... WHERE ... FETCH FIRST 1000 ROWS ONLY`
```

---

## 📊 Impacto Estimado

| Problema | Impacto | Severidade |
|----------|---------|------------|
| Transação não fechada | ⚠️ Esgota pool de conexões | 🔴 CRÍTICO |
| Array `success` infinito | 📈 Uso de RAM linear | 🟡 MÉDIO |
| JSON duplicado | 📈 Uso de RAM | 🟡 MÉDIO |
| Loop N×M×P | 🐌 Lentidão extrema | 🔴 CRÍTICO |
| Timeout longo | ⏱️ Recursos bloqueados | 🟡 MÉDIO |

---

## 🎯 Prioridade de Correção

1. **URGENTE:** Corrigir fechamento de transação
2. **URGENTE:** Otimizar loop N×M×P
3. **ALTA:** Substituir array por contadores
4. **MÉDIA:** Processar em batches
5. **BAIXA:** Ajustar timeouts

---

**Status:** ⚠️ **VAZAMENTOS CRÍTICOS DETECTADOS**  
**Ação:** **CORREÇÃO IMEDIATA NECESSÁRIA**
