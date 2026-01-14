# 🔍 Tracing Jaeger - IDs Registrados

**Data:** 14 de Janeiro de 2026  
**Status:** ✅ Implementado e testado

---

## 📋 Resumo

Sistema de rastreabilidade distribuída implementado com **OpenTelemetry + Jaeger** para monitorar o fluxo completo de integração de produtos e redes.

---

## 🎯 IDs Rastreados no Jaeger

### 1️⃣ **ProductIntegrationUseCase - ImportProductIntegration()**

#### Span Principal: `ImportProductIntegration`

**Atributos Globais:**
```go
total_products              // Total de produtos a processar
final.success_count         // Total de sucessos
final.failure_count         // Total de falhas
final.success_rate          // Taxa de sucesso (%)
```

#### Span de Batch: `ProcessProductBatch`

**Atributos por Batch:**
```go
batch.start                 // Índice inicial do batch (ex: 1)
batch.end                   // Índice final do batch (ex: 100)
batch.size                  // Tamanho do batch (ex: 100)
batch.success_count         // Sucessos acumulados até o batch
batch.failure_count         // Falhas acumuladas até o batch
```

#### Span de Produto: `ProcessSingleProduct`

**Atributos por Produto:** 🔍
```go
staging_id                  // ID da tabela INTEGRACAOPRODUTOSTAGING
product_id                  // ID do produto (IDPRODUTO)
dealer_id                   // ID do revendedor (IDREVENDEDOR)
product_index               // Índice do produto na lista (1, 2, 3...)
processing.success          // true/false - sucesso do processamento
processing.message          // Mensagem de resultado
```

**Eventos Especiais:**
```go
panic.recovered             // Captura de panic com detalhes
  ├─ panic_value            // Valor do panic
  └─ staging_id             // ID do registro que causou panic
```

**Status Codes:**
- ✅ `codes.Ok (1)` - Produto processado com sucesso
- ❌ `codes.Error (2)` - Falha no processamento ou panic

---

### 2️⃣ **IntegrationJobUseCase - ProductNetworkMain()**

#### Span Principal: `ProductNetworkMain`

**Atributos Globais:**
```go
data_corte                  // Data de corte do job (RFC3339)
```

**Eventos de Jobs:**
```go
integration_job.completed          // Job de integração concluído
integration_job.failed             // Job de integração falhou
  └─ error                         // Mensagem de erro

replicate_network.completed        // Replicação de rede concluída
replicate_network.failed           // Replicação falhou
  └─ error                         // Mensagem de erro

move_data.completed                // Movimentação de dados concluída
move_data.failed                   // Movimentação falhou
  └─ error                         // Mensagem de erro

update_expiration.completed        // Atualização de expiração concluída
update_expiration.failed           // Atualização falhou
  └─ error                         // Mensagem de erro
```

**Status Final:**
- ✅ `codes.Ok (1)` - Todos os jobs completados
- ❌ `codes.Error (2)` - Falha em algum job

---

### 3️⃣ **IntegrationJobUseCase - ReplicateNetworkProductsJob()**

#### Span Principal: `ReplicateNetworkProductsJob`

**Atributos Globais:**
```go
total_networks              // Total de redes a processar
```

#### Span de Rede: `ProcessNetwork`

**Atributos por Rede:** 🔍
```go
network_id                  // ID da rede (IDREDE)
dealer_id                   // ID do revendedor principal
network_index               // Índice da rede (1, 2, 3...)
dealers_count               // Quantidade de lojas da rede
```

**Eventos de Rede:**
```go
replicate_products.completed       // Produtos replicados
replicate_products.failed          // Falha ao replicar
  └─ error                         // Mensagem de erro

list_dealers.failed                // Falha ao listar lojas
  └─ error                         // Mensagem de erro

batch_processing.completed         // Processamento batch concluído
  └─ dealers_processed             // Quantidade de lojas processadas

batch_processing.failed            // Falha no processamento batch
  └─ error                         // Mensagem de erro

fallback_processing.completed      // Processamento individual (fallback)
```

**Status por Rede:**
- ✅ `codes.Ok (1)` - Rede processada com sucesso
- ❌ `codes.Error (2)` - Falha no processamento

**Status Final:**
- ✅ `codes.Ok (1)` - Todas as redes replicadas

---

## 🛠️ Como Usar o Jaeger

### 1. Configurar Variáveis de Ambiente

```bash
# .env
JAEGER_ENDPOINT=http://localhost:14268/api/traces
TRACING_ENABLED=true
TRACING_SAMPLE_RATE=1.0  # 1.0 = 100% das requisições
```

### 2. Iniciar Jaeger (Docker)

```bash
docker run -d --name jaeger \
  -e COLLECTOR_ZIPKIN_HOST_PORT=:9411 \
  -p 5775:5775/udp \
  -p 6831:6831/udp \
  -p 6832:6832/udp \
  -p 5778:5778 \
  -p 16686:16686 \
  -p 14268:14268 \
  -p 14250:14250 \
  -p 9411:9411 \
  jaegertracing/all-in-one:latest
```

### 3. Acessar UI do Jaeger

```
http://localhost:16686
```

---

## 🔎 Exemplos de Busca no Jaeger

### Buscar Produto Específico

1. Acesse Jaeger UI
2. **Service:** integracaocron
3. **Operation:** ProcessSingleProduct
4. **Tags:** 
   - `product_id=12345` ou
   - `staging_id=67890` ou
   - `dealer_id=999`

### Buscar Rede Específica

1. **Service:** integracaocron
2. **Operation:** ProcessNetwork
3. **Tags:** `network_id=10`

### Buscar Erros/Panics

1. **Service:** integracaocron
2. **Tags:** `error=true`
3. Ou procure por status `ERROR` na timeline

### Buscar por Período

1. **Lookback:** Custom Time Range
2. Defina data/hora início e fim
3. Filtre por operação desejada

---

## 📊 Visualizações Úteis

### 1. Flamegraph (Timeline)

Mostra a hierarquia de spans:
```
ImportProductIntegration (10min)
├─ ProcessProductBatch (5min)
│  ├─ ProcessSingleProduct (3s) ✅ staging_id=1, product_id=100
│  ├─ ProcessSingleProduct (2s) ✅ staging_id=2, product_id=101
│  └─ ProcessSingleProduct (5s) ❌ staging_id=3, product_id=102 [ERROR]
└─ ProcessProductBatch (5min)
   └─ ...
```

### 2. Trace Graph

Mostra dependências entre serviços e operações.

### 3. Trace Timeline

Visualização temporal detalhada de cada span.

---

## 🎯 Casos de Uso

### Caso 1: Debugar Produto Específico que Falhou

**Problema:** Produto ID 12345 falhou no processamento

**Solução:**
1. Jaeger UI → Search
2. Tags: `product_id=12345`
3. Clique no trace
4. Veja o span `ProcessSingleProduct`
5. Atributos mostrarão:
   - `processing.success = false`
   - `processing.message = "Erro ao chamar procedure Oracle"`
   - `staging_id = 67890`
   - `dealer_id = 999`

### Caso 2: Investigar Panic

**Problema:** Sistema teve panic durante processamento

**Solução:**
1. Jaeger UI → Search
2. Procure eventos: `panic.recovered`
3. Veja atributos:
   - `panic_value = "runtime error: invalid memory address"`
   - `staging_id = 45678`
4. Investigue o registro na tabela INTEGRACAOPRODUTOSTAGING

### Caso 3: Analisar Performance de Batch

**Problema:** Batches estão demorando muito

**Solução:**
1. Busque span: `ProcessProductBatch`
2. Compare duração entre batches
3. Identifique batches lentos
4. Veja quais produtos dentro do batch demoraram mais

### Caso 4: Rastrear Rede Problemática

**Problema:** Rede 10 sempre falha

**Solução:**
1. Busque: `network_id=10`
2. Operation: `ProcessNetwork`
3. Veja eventos:
   - `replicate_products.failed`?
   - `batch_processing.failed`?
4. Analise `dealers_count` - muitas lojas?

---

## 📈 Métricas Extraídas

Com os IDs rastreados, você pode:

1. **Taxa de Sucesso por Produto**
   - Quantos % de produtos são processados com sucesso?

2. **Produtos Problemáticos**
   - Quais `product_id` falham frequentemente?

3. **Revendedores Problemáticos**
   - Quais `dealer_id` causam mais erros?

4. **Redes com Melhor Performance**
   - Quais `network_id` processam mais rápido?

5. **Tempo Médio por Batch**
   - Quanto tempo leva cada batch de 100 produtos?

6. **Panics Recorrentes**
   - Quais `staging_id` causam panics?

---

## 🔧 Correlação com Logs

Os IDs do Jaeger aparecem também nos logs:

```log
2026-01-14 10:30:15 [trace_id=abc123] [span_id=def456] Processing product 1/100 (Staging ID: 67890, Product ID: 12345, Dealer ID: 999)
```

**Trace ID** e **Span ID** permitem:
- Buscar no Jaeger pelo trace exato
- Correlacionar logs com traces visuais
- Debug de ponta a ponta

---

## 🚀 Benefícios

### Antes (Sem Tracing)
❌ "Produto falhou" - qual produto?  
❌ "Rede demorou muito" - qual rede?  
❌ "Panic no batch" - qual registro?  
❌ Logs desconexos e difíceis de correlacionar  

### Depois (Com Tracing)
✅ **Produto ID 12345** falhou (ver trace completo)  
✅ **Rede 10** com **50 lojas** demorou 10min  
✅ **Staging ID 67890** causou panic (memória)  
✅ Timeline visual mostrando gargalos  
✅ Correlação automática entre logs e traces  

---

## 📚 Documentação Relacionada

- [OpenTelemetry Go Docs](https://opentelemetry.io/docs/instrumentation/go/)
- [Jaeger Documentation](https://www.jaegertracing.io/docs/)
- [pkg/tracing/tracing.go](./pkg/tracing/tracing.go) - Funções helper

---

## ✅ Checklist de Validação

- [x] ✅ Context propagado em todas as funções principais
- [x] ✅ IDs críticos registrados (staging_id, product_id, dealer_id, network_id)
- [x] ✅ Eventos de sucesso/falha registrados
- [x] ✅ Panics capturados e registrados
- [x] ✅ Status codes definidos (Ok/Error)
- [x] ✅ Spans hierarquizados (Main → Batch → Product)
- [x] ✅ Atributos numéricos para métricas (success_count, etc.)
- [x] ✅ Código compilado sem erros
- [x] ✅ Tabela de logs corrigida (LogIntegrRMS → LogsIntegrRMS)

---

**🎉 Sistema completamente instrumentado com Jaeger para observabilidade total!**

*Última atualização: 14/01/2026*
