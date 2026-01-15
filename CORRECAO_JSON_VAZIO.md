# 🛡️ Correção: Validação de JSON Vazio

**Data:** 15 de Janeiro de 2026  
**Problema:** `ORA-20000: Erro: ORA-20000: JSON vazio` ao executar `pkg_integra_produto.prc_integra_hermes`  
**Status:** ✅ CORRIGIDO

---

## 🔍 Análise do Problema

### Erro Original
```
ERROR executing pkg_integra_produto.prc_integra_hermes for IPR_ID 108385: 
ORA-20000: Erro: ORA-20000: JSON vazio. - ORA-06512: at "USR_HERMES.PKG_INTEGRA_PRODUTO"
```

### Causa Raiz

A **procedure Oracle** `pkg_integra_produto.prc_integra_hermes` espera:
1. Receber um `IPR_ID` válido
2. Encontrar o registro em `INTEGRRMSPRODUTOIN`
3. **Campo `JSON` preenchido com dados válidos**

Quando o campo `JSON` está vazio ou NULL, a procedure lança `ORA-20000: JSON vazio`.

### Fluxo TypeScript (Referência)

```typescript
// TypeScript valida implicitamente ao fazer JSON.parse()
const produto: IProductInJson = JSON.parse(rms.JSON)
// Se JSON estiver vazio, parse falha e vai para catch

const result = await dopkg_produto(rms.IPR_ID ?? 0)
```

---

## ✅ Solução Implementada

### 1️⃣ Validação Antecipada no Repository

**Arquivo:** `domain/repositories/productIntegrationRepo.go`

```go
func (r *ProductIntegrationRepository) DoPackageProductIntegration(iprID int) (*entities.LogValidate, error) {
    log.Printf("🔍 Validating IPR_ID %d before calling Oracle procedure", iprID)

    // 🛡️ VALIDAÇÃO CRÍTICA: Verificar se o JSON existe e não está vazio
    var jsonData sql.NullString
    checkQuery := `SELECT JSON FROM INTEGRRMSPRODUTOIN WHERE IPR_ID = :1`
    err := r.db.QueryRow(checkQuery, iprID).Scan(&jsonData)
    
    if err == sql.ErrNoRows {
        // Registro não encontrado - pode ser product_id direto
        log.Printf("⚠️ IPR_ID %d not found in INTEGRRMSPRODUTOIN", iprID)
    } else if err != nil {
        log.Printf("❌ ERROR checking JSON for IPR_ID %d: %v", iprID, err)
        return &entities.LogValidate{
            Success: false,
            Message: fmt.Sprintf("Erro ao verificar JSON: %v", err),
        }, nil
    } else {
        // ✅ Registro encontrado - validar JSON
        if !jsonData.Valid || strings.TrimSpace(jsonData.String) == "" {
            msg := fmt.Sprintf("JSON vazio para IPR_ID %d - não é possível processar", iprID)
            log.Printf("❌ VALIDATION FAILED: %s", msg)
            
            return &entities.LogValidate{
                Success: false,
                Message: msg,
            }, nil
        }
        log.Printf("✅ JSON validation passed for IPR_ID %d (length: %d bytes)", 
                   iprID, len(jsonData.String))
    }

    // Chamar procedure Oracle somente se validação passar
    query := `BEGIN pkg_integra_produto.prc_integra_hermes(:1); END;`
    // ...
}
```

### 2️⃣ Validação Adicional no UseCase

**Arquivo:** `domain/usecases/productIntegrationUseCase.go`

```go
func (uc *ProductIntegrationUseCase) processProductIntegration(rms entities.IntegrRmsProductIn) *entities.LogValidate {
    ctx := context.Background()
    ctx, span := tracing.StartSpan(ctx, "ProcessProductIntegration")
    defer span.End()

    // 🔍 Tracing: Registrar IPR_ID
    if rms.IprID != nil {
        tracing.AddInt64Attribute(ctx, "ipr_id", int64(*rms.IprID))
    }

    // ✅ Validar se JSON não está vazio
    if strings.TrimSpace(rms.JSON) == "" {
        msg := "JSON vazio - não é possível processar"
        log.Printf("❌ ERROR: %s for IPR_ID %v", msg, rms.IprID)
        tracing.AddEvent(ctx, "validation.failed", tracing.StringAttr("reason", "empty_json"))
        tracing.SetStatus(ctx, 2, msg)
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
        return &entities.LogValidate{
            Success: false,
            Message: fmt.Sprintf("Error parsing JSON: %v", err),
        }
    }

    // Chamar procedure Oracle
    result, err := uc.repo.DoPackageProductIntegration(*rms.IprID)
    // ...
}
```

### 3️⃣ Tabela de Logs Corrigida

```go
// ANTES (errado)
query := `INSERT INTO LOG_INTEGR_RMS (...) VALUES (...)`

// DEPOIS (correto)
query := `INSERT INTO LOGS_INTEGR_RMS (...) VALUES (...)`
```

---

## 🎯 Benefícios

### Antes da Correção ❌
```
1. Ler IPR_ID 108385 de INTEGRRMSPRODUTOIN
2. JSON está vazio (NULL ou "")
3. Chamar pkg_integra_produto.prc_integra_hermes(108385)
4. Procedure lê JSON vazio
5. ❌ ORA-20000: JSON vazio
6. Aplicação recebe erro
```

### Depois da Correção ✅
```
1. Ler IPR_ID 108385 de INTEGRRMSPRODUTOIN
2. 🛡️ Validação: Verificar se JSON existe
3. 🛡️ Validação: Verificar se JSON não está vazio
4. ❌ JSON vazio detectado
5. ✅ Retornar erro amigável sem chamar procedure
6. 📝 Log detalhado: "JSON vazio para IPR_ID 108385"
7. 🔍 Jaeger registra: validation.failed (reason: empty_json)
```

---

## 📊 Logs Melhorados

### Antes
```
Executing Oracle procedure: pkg_integra_produto.prc_integra_hermes with IPR_ID: 108385
ERROR executing pkg_integra_produto.prc_integra_hermes for IPR_ID 108385: ORA-20000: JSON vazio
```

### Depois
```
🔍 Validating IPR_ID 108385 before calling Oracle procedure
❌ VALIDATION FAILED: JSON vazio para IPR_ID 108385 - não é possível processar
```

---

## 🔍 Rastreamento Jaeger

### Span: `ProcessProductIntegration`

**Atributos:**
- `ipr_id = 108385`

**Eventos:**
- `validation.failed`
  - `reason = "empty_json"`

**Status:**
- `ERROR: JSON vazio - não é possível processar`

**Benefício:** Permite buscar no Jaeger por `ipr_id=108385` e ver exatamente o que aconteceu.

---

## 🧪 Como Testar

### 1. Cenário: JSON Vazio

```sql
-- Inserir registro com JSON vazio
INSERT INTO INTEGRRMSPRODUTOIN (IPR_ID, JSON, DATARECEBIMENTO)
VALUES (999999, '', SYSDATE);
```

**Resultado Esperado:**
```
🔍 Validating IPR_ID 999999 before calling Oracle procedure
❌ VALIDATION FAILED: JSON vazio para IPR_ID 999999 - não é possível processar
```

### 2. Cenário: JSON NULL

```sql
INSERT INTO INTEGRRMSPRODUTOIN (IPR_ID, JSON, DATARECEBIMENTO)
VALUES (999998, NULL, SYSDATE);
```

**Resultado Esperado:**
```
🔍 Validating IPR_ID 999998 before calling Oracle procedure
❌ VALIDATION FAILED: JSON vazio para IPR_ID 999998 - não é possível processar
```

### 3. Cenário: JSON Válido

```sql
INSERT INTO INTEGRRMSPRODUTOIN (IPR_ID, JSON, DATARECEBIMENTO)
VALUES (999997, '{"cod":"123","desc":"Produto Teste"}', SYSDATE);
```

**Resultado Esperado:**
```
🔍 Validating IPR_ID 999997 before calling Oracle procedure
✅ JSON validation passed for IPR_ID 999997 (length: 45 bytes)
📞 Executing Oracle procedure: pkg_integra_produto.prc_integra_hermes with IPR_ID: 999997
✅ Oracle procedure executed successfully for IPR_ID 999997
```

---

## 📋 Checklist de Validação

- [x] ✅ Validação de JSON vazio no repository
- [x] ✅ Validação de JSON vazio no usecase
- [x] ✅ Mensagem de erro amigável
- [x] ✅ Logs detalhados com emojis
- [x] ✅ Tracing Jaeger com IPR_ID
- [x] ✅ Evita chamada à procedure se JSON vazio
- [x] ✅ Tabela de logs corrigida (LOGS_INTEGR_RMS)
- [x] ✅ Import `strings` adicionado
- [x] ✅ Código compilado sem erros

---

## 🚀 Próximos Passos

### 1. Monitoramento
- Verificar quantos registros com JSON vazio existem:
  ```sql
  SELECT COUNT(*) FROM INTEGRRMSPRODUTOIN 
  WHERE JSON IS NULL OR TRIM(JSON) = '';
  ```

### 2. Limpeza
- Remover registros inválidos:
  ```sql
  DELETE FROM INTEGRRMSPRODUTOIN 
  WHERE JSON IS NULL OR TRIM(JSON) = '';
  ```

### 3. Prevenção
- Adicionar constraint no banco:
  ```sql
  ALTER TABLE INTEGRRMSPRODUTOIN
  ADD CONSTRAINT chk_json_not_empty 
  CHECK (JSON IS NOT NULL AND LENGTH(TRIM(JSON)) > 0);
  ```

---

## 📚 Arquivos Modificados

1. ✅ `domain/repositories/productIntegrationRepo.go`
   - Validação de JSON vazio antes de chamar procedure
   - Tabela de logs corrigida

2. ✅ `domain/usecases/productIntegrationUseCase.go`
   - Validação adicional de JSON vazio
   - Tracing com IPR_ID
   - Import `strings` adicionado

---

**🎉 Problema resolvido! Agora o sistema valida JSON vazio ANTES de chamar a procedure Oracle, evitando o erro ORA-20000.**

*Última atualização: 15/01/2026*
