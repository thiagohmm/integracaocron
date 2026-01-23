# Resumo das Alterações - Variável PORT e Health Check

## Data: 2026-01-23

## Objetivos

1. Alterar a configuração do servidor HTTP Gin para usar a variável de ambiente `PORT` ao invés de `HTTP_PORT`, mantendo compatibilidade retroativa.
2. Melhorar o endpoint `/health` para verificar a conectividade real com banco de dados Oracle e RabbitMQ.

## Arquivos Modificados

### 1. configuration/config.go

**Alterações:**

- Adicionado campo `HTTPPortLegacy` para manter compatibilidade com `HTTP_PORT`
- Campo `HTTPPort` agora mapeia para `PORT` (variável primária)
- Ambas as variáveis são carregadas do arquivo .env ou variáveis de ambiente

**Código:**

```go
type Conf struct {
    // ...
    HTTPPort       string `mapstructure:"PORT"`        // Primary: PORT variable
    HTTPPortLegacy string `mapstructure:"HTTP_PORT"`   // Backward compatibility
    // ...
}
```

### 2. cmd/app/main.go

**Alterações:**

- Função `getHTTPPort()` atualizada com lógica de prioridade em cascata
- Prioridade de busca:
  1. Variável de ambiente `PORT`
  2. Variável de ambiente `HTTP_PORT` (compatibilidade)
  3. `PORT` do arquivo .env
  4. `HTTP_PORT` do arquivo .env (compatibilidade)
  5. Valor padrão "8080"

**Código:**

```go
func getHTTPPort(cfg *configuration.Conf) string {
    // First priority: PORT environment variable
    httpPort := os.Getenv("PORT")
    if httpPort != "" {
        return httpPort
    }

    // Second priority: HTTP_PORT environment variable (backward compatibility)
    httpPort = os.Getenv("HTTP_PORT")
    if httpPort != "" {
        return httpPort
    }

    // Third priority: PORT from config file
    if cfg.HTTPPort != "" {
        return cfg.HTTPPort
    }

    // Fourth priority: HTTP_PORT from config file (backward compatibility)
    if cfg.HTTPPortLegacy != "" {
        return cfg.HTTPPortLegacy
    }

    // Default fallback
    return "8080"
}
```

### 3. internal/delivery/http_server.go

**Alterações:**

- Adicionado parâmetros `db *sql.DB` e `rabbitmqURL string` ao construtor
- Implementado health check completo com verificações de:
  - Conexão com banco de dados Oracle (ping com timeout de 5s)
  - Conexão com RabbitMQ (dial + channel com timeout de 5s)
- Retorna status HTTP 503 quando algum serviço está indisponível
- Resposta JSON detalhada com status de cada componente

**Estruturas Adicionadas:**

```go
type HealthCheckResponse struct {
    Status    string                 `json:"status"`
    Timestamp string                 `json:"timestamp"`
    Checks    map[string]HealthCheck `json:"checks"`
}

type HealthCheck struct {
    Status  string `json:"status"`
    Message string `json:"message,omitempty"`
}
```

### 4. cmd/app/main.go

**Alterações:**

- Atualizada chamada `NewHTTPServer()` para passar `db` e `rabbitmqURL`
- Permite que o health check tenha acesso às conexões para verificação

### 5. README.md

**Alterações:**

- Documentação atualizada para refletir o uso de `PORT`
- Adicionada nota sobre compatibilidade com `HTTP_PORT`
- Exemplo de .env atualizado com `PORT=8080`
- Documentação completa do endpoint `/health` com exemplos de respostas
- Descrição das verificações realizadas e códigos HTTP retornados

## Compatibilidade Retroativa

✅ **Totalmente compatível** com configurações existentes que usam `HTTP_PORT`

### Cenários de Uso:

1. **Novo padrão (recomendado):**

   ```bash
   PORT=8080
   ```

2. **Padrão antigo (ainda funciona):**

   ```bash
   HTTP_PORT=8080
   ```

3. **Ambos definidos (PORT tem prioridade):**

   ```bash
   PORT=3000
   HTTP_PORT=8080  # Será ignorado
   ```

   Resultado: Servidor rodará na porta 3000

4. **Nenhum definido:**
   Resultado: Servidor rodará na porta padrão 8080

## Benefícios

1. **Padrão da Indústria:** `PORT` é a convenção mais comum em aplicações web
2. **Compatibilidade com Plataformas:** Muitas plataformas de deploy (Heroku, Railway, etc.) usam `PORT`
3. **Retrocompatibilidade:** Sistemas existentes continuam funcionando sem alterações
4. **Flexibilidade:** Múltiplas formas de configurar a porta

## Testes Recomendados

1. Testar com `PORT` definido no .env
2. Testar com `HTTP_PORT` definido no .env (compatibilidade)
3. Testar sem nenhuma variável (deve usar 8080)
4. Testar com variável de ambiente sobrescrevendo .env

## Exemplo de Uso

```bash
# No arquivo .env
PORT=3000

# Ou via linha de comando
PORT=3000 ./bin/integracaocron

# Ou via Docker
docker run -e PORT=3000 integracaocron
```

## Logs Esperados

```
Iniciando servidor HTTP na porta 3000
```

## Health Check Melhorado

### Endpoint: GET /health

**Verificações Realizadas:**

1. **Database (Oracle):**
   - Executa `PingContext()` com timeout de 5 segundos
   - Verifica se a conexão está ativa e responsiva
   - **Recuperação Automática:**
     - Detecta conexões quebradas (broken pipe, connection reset, EOF, bad connection)
     - Quando detecta falha, tenta um segundo ping (retry) com timeout de 3 segundos
     - Usa o pool de conexões do Go para recuperação automática
     - Retorna "healthy" se conseguir reconectar, "unhealthy" caso contrário

2. **RabbitMQ:**
   - Tenta estabelecer conexão com `amqp.Dial()`
   - Abre um canal para verificar funcionalidade completa
   - Timeout de 5 segundos para toda a operação

**Respostas:**

✅ **Sucesso (200 OK):**

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

❌ **Falha (503 Service Unavailable):**

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

### Benefícios do Health Check Melhorado

1. **Monitoramento Real:** Verifica conectividade real, não apenas se o serviço está rodando
2. **Recuperação Automática:** Detecta e tenta recuperar conexões quebradas automaticamente
3. **Kubernetes/Docker Ready:** Compatível com liveness e readiness probes
4. **Debugging Facilitado:** Identifica exatamente qual componente está com problema
5. **Timeout Configurado:** Evita que health checks travem indefinidamente
6. **Logs Detalhados:** Erros são logados para troubleshooting com níveis apropriados (Warn, Error, Info)
7. **Pool de Conexões:** Aproveita o pool de conexões do Go para recuperação eficiente

## Notas Importantes

### Variável PORT

- A mudança é **não-destrutiva** - código antigo continua funcionando
- Recomenda-se migrar para `PORT` em novos deployments
- `HTTP_PORT` será mantido indefinidamente para compatibilidade

### Health Check

- Ideal para uso em orquestradores (Kubernetes, Docker Swarm, etc.)
- Pode ser usado para alertas de monitoramento
- Timeout de 5 segundos por verificação evita travamentos
- Recuperação automática de conexões quebradas
- Logs informativos para troubleshooting:
  - `WARN`: Quando detecta conexão quebrada e tenta recuperar
  - `INFO`: Quando consegue recuperar a conexão
  - `ERROR`: Quando falha após tentativa de recuperação
