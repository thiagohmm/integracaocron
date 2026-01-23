# IntegracaoCron - Sistema de Integração

Sistema de integração desenvolvido em Go para processamento de dados via RabbitMQ com Oracle Database.

## 🚀 Características

- **Processamento assíncrono** via RabbitMQ
- **Workers concorrentes** configuráveis
- **Integração Oracle Database**
- **Jobs de limpeza e manutenção** automáticos
- **Graceful shutdown**
- **Logging detalhado**
- **Containerização Docker**
- **Tracing distribuído** com Jaeger
- **UUID para rastreamento** de transações

## 📋 Pré-requisitos

- Go 1.21+
- Oracle Database
- RabbitMQ
- Docker (opcional)

## 🛠️ Instalação e Configuração

### 1. Clone o repositório

```bash
git clone https://github.com/thiagohmm/integracaocron.git
cd integracaocron
```

### 2. Configure as variáveis de ambiente

Crie um arquivo `.env` na raiz do projeto:

```bash
# Database Configuration
DB_DIALECT=oracle
DB_USER=seu_usuario_db
DB_PASSWD=sua_senha_db
DB_SCHEMA=seu_schema
DB_CONNECTSTRING=host=localhost port=1521 service_name=ORCL

# RabbitMQ Configuration
ENV_RABBITMQ=amqp://usuario:senha@localhost:5672/

# Redis Configuration (se necessário)
ENV_REDIS_ADDRESS=localhost:6379
ENV_REDIS_PASSWORD=
ENV_REDIS_EXPIRE=3600

# Application Configuration
WORKERS=20
PORT=8080

# Jaeger Tracing
JAEGER_ENDPOINT=http://localhost:14268/api/traces
TRACING_ENABLED=true
TRACING_SAMPLE_RATE=1.0
```

### 3. Instale as dependências

```bash
go mod tidy
```

## 🏃 Executando a Aplicação

### Opção 1: Executar diretamente

```bash
# Usar o Makefile
make run

# Ou usar o script
./run.sh

# Ou compilar e executar manualmente
go build -o bin/integracaocron ./cmd/app/
./bin/integracaocron
```

### Opção 2: Docker

```bash
# Construir e executar com Docker Compose
make docker-run

# Ou manualmente
docker build -t integracaocron .
docker run --env-file .env integracaocron
```

### Opção 3: Docker Compose (com RabbitMQ)

```bash
# Subir toda a stack (app + RabbitMQ)
docker-compose up -d

# Ver logs
docker-compose logs -f integracaocron

# Parar
docker-compose down
```

## 📊 Comandos Makefile

```bash
make help          # Mostra todos os comandos disponíveis
make build         # Compila a aplicação
make run           # Compila e executa
make test          # Executa testes
make clean         # Limpa arquivos de build
make docker-build  # Constrói imagem Docker
make docker-run    # Executa com Docker Compose
make docker-stop   # Para containers Docker
make dev           # Workflow de desenvolvimento (fmt + vet + test + build)
make prod-build    # Build otimizado para produção
```

## 🏗️ Arquitetura

```
cmd/
├── app/
│   └── main.go                 # Ponto de entrada da aplicação

domain/
├── entities/                   # Entidades de domínio
├── repositories/               # Interfaces de repositórios
└── usecases/                   # Casos de uso de negócio

infraestructure/
├── database/                   # Implementação Oracle
└── rabbitmq/                   # Implementação RabbitMQ

internal/
└── delivery/                   # Handlers e listeners
```

## 🔄 Fluxo de Processamento

1. **Listener RabbitMQ** recebe mensagens da fila `integracaoCron`
2. **Workers concorrentes** processam as mensagens
3. **Use Cases** executam a lógica de negócio específica
4. **Repositories** fazem as operações no banco de dados
5. **Integration Job** executa limpeza e manutenção automática
6. **Logs** são enviados para fila de auditoria

## 📝 Tipos de Integração Suportados

- **Promoção** (`tipoIntegracao: "Promocao"`)
- **Estrutura Mercadológica** (`tipoIntegracao: "EstruturaMercadologica"`)
- **Produtos** (`tipoIntegracao: "Produtos"`)

## 🔍 Monitoramento e Observabilidade

### Jaeger Tracing

A aplicação suporta distributed tracing com Jaeger para monitoramento e debugging.

#### Configuração do Jaeger

```bash
# No arquivo .env
JAEGER_ENDPOINT=http://localhost:14268/api/traces
TRACING_ENABLED=true
TRACING_SAMPLE_RATE=1.0
```

#### Executar Jaeger com Docker

```bash
# Subir apenas o Jaeger
docker run -d --name jaeger \
  -p 16686:16686 \
  -p 14268:14268 \
  jaegertracing/all-in-one:latest

# Ou usar o docker-compose completo
docker-compose -f docker-compose.jaeger.yml up -d
```

#### Acessar Jaeger UI

- URL: http://localhost:16686
- Visualizar traces da aplicação `integracaocron`
- Analisar performance e logs integrados

### UUID para Rastreamento

Cada transação de mover produto e promoção gera um UUID único que é:

- Adicionado ao Jaeger como atributo (`transaction_uuid`, `uuid`)
- Enviado para a fila "log" no campo `TRANSACAO`
- Exibido nos logs do console
- Propagado através do contexto usando OpenTelemetry Baggage

**Formato do log:**

```
[INFO][trace:5625da8d][span:0cab5369][uuid:550e8400-e29b-41d4-a716-446655440000] Iniciando processamento de produto
```

**Buscar no Jaeger:**

- Tags: `uuid=<seu-uuid>` ou `transaction_uuid=<seu-uuid>`
- Operation: `MoverProduto` ou `MoverPromocao`

### Logs da aplicação

```bash
# Docker Compose
docker-compose logs -f integracaocron

# Aplicação local
tail -f logs/integracaocron.log
```

### RabbitMQ Management (se usando Docker Compose)

- URL: http://localhost:15672
- Usuário: admin
- Senha: admin123

## 📡 URLs e Endpoints HTTP

### GET /health

**Porta:** Configurável via variável de ambiente `PORT` (padrão: 8080)

**Finalidade:** Endpoint de verificação de saúde (health check) do serviço. Verifica a conectividade com o banco de dados Oracle e RabbitMQ.

**Resposta de Sucesso (200 OK):**

```json
{
  "status": "healthy",
  "timestamp": "2026-01-23T17:50:00Z",
  "checks": {
    "database": {
      "status": "healthy",
      "message": "Database connection is active"
    },
    "rabbitmq": {
      "status": "healthy",
      "message": "RabbitMQ connection is active"
    }
  }
}
```

**Resposta de Falha (503 Service Unavailable):**

```json
{
  "status": "unhealthy",
  "timestamp": "2026-01-23T17:50:00Z",
  "checks": {
    "database": {
      "status": "unhealthy",
      "message": "Database ping failed: connection refused"
    },
    "rabbitmq": {
      "status": "healthy",
      "message": "RabbitMQ connection is active"
    }
  }
}
```

**Verificações Realizadas:**

- **Database:** Ping no banco de dados Oracle com timeout de 5 segundos
  - Detecta conexões quebradas (broken pipe, connection reset, EOF)
  - Tenta reconectar automaticamente quando detecta falha de conexão
  - Usa o pool de conexões do Go para recuperação automática
- **RabbitMQ:** Tentativa de conexão e abertura de canal com timeout de 5 segundos

**Códigos HTTP:**

- `200 OK` - Todos os serviços estão saudáveis
- `503 Service Unavailable` - Um ou mais serviços estão indisponíveis

### POST /integration

**Porta:** Configurável via variável de ambiente `PORT` (padrão: 8080)

**Nota:** Por compatibilidade, a variável `HTTP_PORT` ainda é suportada, mas `PORT` tem prioridade.

**Finalidade:** Endpoint principal para processar diferentes tipos de integrações

**Formato da Requisição:**

```json
{
  "tipoIntegracao": "tipo_da_integracao",
  "dados": {
    // dados específicos da integração (opcional)
  }
}
```

#### Tipos de Integração Suportados:

##### Promoção (`promocao` ou `Promocao`)

```json
{
  "tipoIntegracao": "promocao",
  "dados": {
    "ipm_id": 12345
  }
}
```

##### Produto (`produto` ou `Produto`)

```json
{
  "tipoIntegracao": "produto"
}
```

##### Normalização de Promoção (`promocao_normalizacao` ou `PromocaoNormalizacao`)

```json
{
  "tipoIntegracao": "promocao_normalizacao"
}
```

##### Product Network Main (`mover`, `productNetworkMain` ou `product_network_main`)

```json
{
  "tipoIntegracao": "mover"
}
```

**Respostas do Endpoint /integration**

**Sucesso (200 OK):**

```json
{
  "success": true,
  "message": "Integração processada com sucesso"
}
```

**Erro de Validação (400 Bad Request):**

```json
{
  "success": false,
  "error": "Invalid request format: [detalhes do erro]"
}
```

**Erro de Processamento (500 Internal Server Error):**

```json
{
  "success": false,
  "error": "[mensagem de erro específica]"
}
```

### Exemplos de Uso com cURL

#### Health Check

```bash
# Verificar saúde do serviço
curl -X GET http://localhost:8080/health

# Com formatação JSON (usando jq)
curl -X GET http://localhost:8080/health | jq

# Verificar apenas o código de status
curl -X GET -o /dev/null -w "%{http_code}" http://localhost:8080/health

# Testar recuperação automática (após religar o banco)
./scripts/test_health_recovery.sh 8080
```

#### Processar Promoção Específica

```bash
curl -X POST http://localhost:8080/integration \
  -H "Content-Type: application/json" \
  -d '{
    "tipoIntegracao": "promocao",
    "dados": {
      "ipm_id": 12345
    }
  }'
```

#### Processar Integração de Produtos

```bash
curl -X POST http://localhost:8080/integration \
  -H "Content-Type: application/json" \
  -d '{
    "tipoIntegracao": "produto"
  }'
```

## ⚙️ Configurações Avançadas

### Número de Workers

```bash
# Via variável de ambiente
export WORKERS=50

# Via .env
WORKERS=50
```

### Timeout de Conexão

As conexões com banco de dados têm timeout de 30 segundos por padrão.

### Graceful Shutdown

A aplicação responde aos sinais SIGTERM e SIGINT para shutdown graceful.

## 🧪 Desenvolvimento

### Executar testes

```bash
make test
```

### Formatar código

```bash
make fmt
```

### Verificar código

```bash
make vet
```

### Workflow completo de desenvolvimento

```bash
make dev  # fmt + vet + test + build
```

## 📊 Estrutura de Mensagens RabbitMQ

### Formato da mensagem de entrada

```json
{
  "tipoIntegracao": "Promocao",
  "dados": {
    "IPMD_ID": 123,
    "Json": "{\"descricao\": \"Promoção teste\"}",
    "DATARECEBIMENTO": "2025-10-06 12:00:00"
  }
}
```

### Formato do log de saída

```json
{
  "tabela": "LogsIntegrRMS",
  "fields": [
    "TRANSACAO",
    "TABELA",
    "DATARECEBIMENTO",
    "DATAPROCESSAMENTO",
    "STATUSPROCESSAMENTO",
    "JSON",
    "DESCRICAOERRO"
  ],
  "values": [
    "IN",
    "PROMOCAO",
    "2025-10-06 12:00:00",
    "2025-10-06 12:05:00",
    0,
    "{...}",
    "Processamento realizado com sucesso."
  ]
}
```

## 🔍 Tracing Jaeger - IDs Registrados

### ProductIntegrationUseCase - ImportProductIntegration()

#### Span Principal: `ImportProductIntegration`

**Atributos Globais:**

- `total_products` - Total de produtos a processar
- `final.success_count` - Total de sucessos
- `final.failure_count` - Total de falhas
- `final.success_rate` - Taxa de sucesso (%)

#### Span de Batch: `ProcessProductBatch`

**Atributos por Batch:**

- `batch.start` - Índice inicial do batch
- `batch.end` - Índice final do batch
- `batch.size` - Tamanho do batch
- `batch.success_count` - Sucessos acumulados
- `batch.failure_count` - Falhas acumuladas

#### Span de Produto: `ProcessSingleProduct`

**Atributos por Produto:**

- `staging_id` - ID da tabela INTEGRACAOPRODUTOSTAGING
- `product_id` - ID do produto (IDPRODUTO)
- `dealer_id` - ID do revendedor (IDREVENDEDOR)
- `product_index` - Índice do produto na lista
- `processing.success` - true/false - sucesso do processamento
- `processing.message` - Mensagem de resultado

### IntegrationJobUseCase - ProductNetworkMain()

#### Span Principal: `ProductNetworkMain`

**Atributos Globais:**

- `data_corte` - Data de corte do job (RFC3339)

**Eventos de Jobs:**

- `integration_job.completed` / `integration_job.failed`
- `replicate_network.completed` / `replicate_network.failed`
- `move_data.completed` / `move_data.failed`
- `update_expiration.completed` / `update_expiration.failed`

### IntegrationJobUseCase - ReplicateNetworkProductsJob()

#### Span Principal: `ReplicateNetworkProductsJob`

**Atributos Globais:**

- `total_networks` - Total de redes a processar

#### Span de Rede: `ProcessNetwork`

**Atributos por Rede:**

- `network_id` - ID da rede (IDREDE)
- `dealer_id` - ID do revendedor principal
- `network_index` - Índice da rede
- `dealers_count` - Quantidade de lojas da rede

### MoverProduto e MoverPromocao

**Atributos:**

- `transaction_uuid` - UUID único da transação
- `uuid` - UUID (alias para facilitar busca)
- `transaction_type` - Tipo: "mover_produto" ou "mover_promocao"
- `last_transaction_uuid` - UUID da última transação processada

**Eventos:**

- `transaction.started` - Início da transação
- `transaction.completed` - Transação concluída

## 🛡️ Correções Aplicadas

### Correção: Validação de JSON Vazio

**Problema:** `ORA-20000: Erro: ORA-20000: JSON vazio` ao executar `pkg_integra_produto.prc_integra_hermes`

**Solução:** Validação antecipada no repository e usecase antes de chamar a procedure Oracle.

**Arquivos Modificados:**

- `domain/repositories/productIntegrationRepo.go`
- `domain/usecases/productIntegrationUseCase.go`

### Correção: Nome das Colunas da Tabela REDE

**Problema:** `ORA-00904: "REPLICAR_PRODUTO": invalid identifier`

**Solução:** Todas as queries corrigidas para usar nomes sem underscore (padrão Oracle).

**Arquivo Modificado:**

- `domain/repositories/networkRepo.go`

### Correção: Processamento de Produtos

**Problema:** Código Go não estava chamando a procedure Oracle `pkg_integra_produto.prc_integra_hermes`

**Solução:** Implementada chamada correta à procedure Oracle, igual ao código TypeScript.

**Arquivo Modificado:**

- `domain/usecases/productIntegrationUseCase.go`

### Correções de Vazamento de Memória

**Problemas Corrigidos:**

1. ✅ Transação sem garantia de fechamento
2. ✅ Array `success []bool` crescendo infinitamente
3. ✅ Processamento sem batches
4. ✅ Duplicação de JSON na memória
5. ✅ Transação em ProductNetworkMain sem flag

**Melhorias:**

- Processamento em batches de 100 produtos
- Uso de contadores ao invés de arrays
- Transações sempre fechadas com pattern `committed := false` + defer
- Eliminação de duplicação de JSON nos logs

**Arquivos Modificados:**

- `domain/usecases/productIntegrationUseCase.go`
- `domain/usecases/integrationJobUseCase.go`

### Correção: Timeout de Stored Procedures

**Problema:** `ORA-01013: user requested cancel` - stored procedures sendo canceladas após 2 minutos

**Solução:** Timeout aumentado de 120 segundos (2 minutos) para 600 segundos (10 minutos) em todas as funções de remoção de transação.

**Arquivo Modificado:**

- `domain/repositories/integrationRepo.go`

### Correção: IDREDE Invalid Identifier

**Problema:** `ORA-00904: "IDREDE": invalid identifier` na tabela `INTEGRACAOPRODUTOSTAGING`

**Solução:** Removida condição `AND IDREDE = :1` da query, pois a tabela não possui essa coluna.

**Arquivo Modificado:**

- `domain/repositories/networkRepo.go`

## 🔄 Fluxo de Processamento de Produtos

### Fluxo Completo

```
Sistema Externo → INTEGRRMSPRODUTOIN (dados brutos)
                    ↓
Trigger/Job executa: pkg_integra_produto.prc_integra_hermes(IPR_ID)
                    ↓
Dados movidos para → INTEGRACAOPRODUTOSTAGING (JSON)
                    ↓
RabbitMQ mensagem "produto" → ImportProductIntegration (Go)
                    ↓
Lê registros da INTEGRACAOPRODUTOSTAGING
                    ↓
Para cada registro:
  ├─ Parse JSON (validação)
  ├─ Chama pkg_integra_produto.prc_integra_hermes(IdProduto)
  ├─ Loga resultado (sucesso/falha)
  └─ Remove da staging (sempre)
                    ↓
Produto integrado no sistema final → PRODUTO
```

## 🐛 Troubleshooting

### Problemas de conexão com Oracle

- Verifique se o Oracle Client está instalado
- Confirme as configurações de conexão no `.env`
- Teste a conectividade com `tnsping`

### Problemas de conexão com RabbitMQ

- Verifique se o RabbitMQ está rodando
- Confirme as credenciais e URL
- Teste com `rabbitmqctl status`

### Aplicação não processa mensagens

- Verifique se a fila `integracaoCron` existe
- Confirme se há mensagens na fila
- Verifique os logs para erros específicos

### JSON Vazio

- Verifique se há registros com JSON vazio na tabela `INTEGRRMSPRODUTOIN`
- O sistema agora valida JSON antes de chamar a procedure Oracle
- Logs mostrarão: "JSON vazio para IPR_ID X - não é possível processar"

### Timeout em Stored Procedures

- Stored procedures de limpeza agora têm timeout de 10 minutos
- Se ainda ocorrer timeout, verifique o volume de dados
- Considere otimizar a stored procedure no Oracle

## 📄 Licença

Este projeto é propriedade privada da empresa.

## 👥 Contribuição

Para contribuir com o projeto, siga o padrão de commits convencionais e abra um Pull Request.

---

**Desenvolvido com ❤️ em Go**
