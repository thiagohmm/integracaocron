# Lista de URLs do Sistema IntegracaoCron

Com base na análise do código, o sistema possui as seguintes URLs/endpoints HTTP:

## 1. GET /health

**Porta:** Configurável via variável de ambiente `HTTP_PORT` (padrão: 8080)

**Finalidade:** Endpoint de verificação de saúde (health check) do serviço

**Descrição:** Retorna o status do serviço para verificar se a aplicação está funcionando corretamente. Útil para monitoramento e load balancers.

**Resposta de Sucesso:**

```json
{
  "status": "ok"
}
```

**Código HTTP:** 200 OK

---

## 2. POST /integration

**Porta:** Configurável via variável de ambiente `HTTP_PORT` (padrão: 8080)

**Finalidade:** Endpoint principal para processar diferentes tipos de integrações

**Descrição:** Processa diversos tipos de integrações de dados, incluindo promoções, produtos e normalizações. O tipo de integração é determinado pelo campo `tipoIntegracao` no corpo da requisição.

**Formato da Requisição:**

```json
{
  "tipoIntegracao": "tipo_da_integracao",
  "dados": {
    // dados específicos da integração (opcional)
  }
}
```

### Tipos de Integração Suportados:

#### 2.1. Promoção (`promocao` ou `Promocao`)

**Finalidade:** Processa promoções individuais ou todas as promoções pendentes

**Comportamento:**

- Se `dados` estiver vazio ou sem `ipm_id`: processa todas as promoções pendentes do banco
- Se `dados` contiver `ipm_id`: processa apenas a promoção específica

**Exemplo com promoção específica:**

```json
{
  "tipoIntegracao": "promocao",
  "dados": {
    "ipm_id": 12345
  }
}
```

**Exemplo para processar todas:**

```json
{
  "tipoIntegracao": "promocao",
  "dados": {}
}
```

#### 2.2. Produto (`produto` ou `Produto`)

**Finalidade:** Importa e processa integrações de produtos

**Descrição:** Executa a importação completa de produtos do sistema de integração

**Exemplo:**

```json
{
  "tipoIntegracao": "produto"
}
```

#### 2.3. Normalização de Promoção (`promocao_normalizacao` ou `PromocaoNormalizacao`)

**Finalidade:** Normaliza dados de promoções existentes

**Descrição:** Processa e normaliza promoções, removendo duplicatas e padronizando informações

**Exemplo:**

```json
{
  "tipoIntegracao": "promocao_normalizacao"
}
```

#### 2.4. Product Network Main (`mover`, `productNetworkMain` ou `product_network_main`)

**Finalidade:** Processa a movimentação de produtos na rede

**Descrição:** Executa o processo de integração de produtos com a rede principal, usando a data atual como data de corte

**Exemplo:**

```json
{
  "tipoIntegracao": "mover"
}
```

### Respostas do Endpoint /integration

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

---

## Configuração do Servidor

**Porta HTTP:** Definida pela variável de ambiente `HTTP_PORT` ou configuração (padrão: 8080)

**URL Base:** `http://localhost:8080` (em ambiente local)

**Exemplos de URLs Completas:**

- Health Check: `http://localhost:8080/health`
- Integração: `http://localhost:8080/integration`

---

## Exemplos de Uso com cURL

### Health Check

```bash
curl -X GET http://localhost:8080/health
```

### Processar Promoção Específica

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

### Processar Todas as Promoções Pendentes

```bash
curl -X POST http://localhost:8080/integration \
  -H "Content-Type: application/json" \
  -d '{
    "tipoIntegracao": "promocao",
    "dados": {}
  }'
```

### Processar Integração de Produtos

```bash
curl -X POST http://localhost:8080/integration \
  -H "Content-Type: application/json" \
  -d '{
    "tipoIntegracao": "produto"
  }'
```

### Normalizar Promoções

```bash
curl -X POST http://localhost:8080/integration \
  -H "Content-Type: application/json" \
  -d '{
    "tipoIntegracao": "promocao_normalizacao"
  }'
```

### Processar Product Network Main

```bash
curl -X POST http://localhost:8080/integration \
  -H "Content-Type: application/json" \
  -d '{
    "tipoIntegracao": "mover"
  }'
```

---

## Observações Importantes

1. **Tracing:** Todos os endpoints possuem rastreamento distribuído (OpenTelemetry/Jaeger) habilitado
2. **Logging:** Todas as requisições são logadas com contexto detalhado
3. **Middleware:** O servidor utiliza Gin framework com middleware de logging e recovery
4. **Modo:** O servidor roda em modo release (produção)
5. **Workers:** O sistema também consome mensagens do RabbitMQ com número configurável de workers (padrão: 20)
6. **Graceful Shutdown:** O servidor suporta desligamento gracioso através de sinais SIGTERM/SIGINT

---

## Variáveis de Ambiente

| Variável              | Descrição                                  | Padrão                            |
| --------------------- | ------------------------------------------ | --------------------------------- |
| `HTTP_PORT`           | Porta do servidor HTTP                     | 8080                              |
| `ENV_RABBITMQ`        | URL de conexão do RabbitMQ                 | -                                 |
| `WORKERS`             | Número de workers para processar mensagens | 20                                |
| `JAEGER_ENDPOINT`     | Endpoint do Jaeger para tracing            | http://localhost:14268/api/traces |
| `TRACING_ENABLED`     | Habilita/desabilita tracing                | true                              |
| `TRACING_SAMPLE_RATE` | Taxa de amostragem do tracing              | 1.0                               |

---

## Arquitetura

O sistema utiliza:

- **Framework Web:** Gin (Go)
- **Banco de Dados:** Oracle
- **Message Broker:** RabbitMQ
- **Tracing:** OpenTelemetry + Jaeger
- **Padrão:** Clean Architecture (Domain, Use Cases, Infrastructure, Delivery)
