# Guia de Teste - Health Check com Recuperação Automática

## Data: 2026-01-23

## Objetivo

Validar que o endpoint `/health` detecta e recupera automaticamente de falhas de conexão com o banco de dados Oracle.

## Cenários de Teste

### Cenário 1: Sistema Saudável

**Pré-condições:**

- Banco de dados Oracle está online e acessível
- RabbitMQ está online e acessível
- Aplicação está rodando

**Teste:**

```bash
curl -X GET http://localhost:8080/health | jq
```

**Resultado Esperado:**

```json
{
  "status": "healthy",
  "timestamp": "2026-01-23T17:57:59-03:00",
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

**Código HTTP:** 200 OK

---

### Cenário 2: Banco de Dados Desligado

**Pré-condições:**

- Banco de dados Oracle está OFFLINE
- RabbitMQ está online
- Aplicação está rodando

**Teste:**

```bash
curl -X GET http://localhost:8080/health | jq
```

**Resultado Esperado:**

```json
{
  "status": "unhealthy",
  "timestamp": "2026-01-23T17:57:59-03:00",
  "checks": {
    "database": {
      "status": "unhealthy",
      "message": "Database ping failed: write tcp 192.168.230.248:50696->10.93.10.234:1522: write: broken pipe"
    },
    "rabbitmq": {
      "status": "healthy",
      "message": "RabbitMQ connection is active"
    }
  }
}
```

**Código HTTP:** 503 Service Unavailable

**Logs Esperados:**

```
[WARN] Database connection broken, attempting to reconnect: write tcp ... broken pipe
[ERROR] Database health check failed after retry: ...
```

---

### Cenário 3: Recuperação Automática (TESTE PRINCIPAL)

**Pré-condições:**

- Banco de dados Oracle foi desligado e depois religado
- RabbitMQ está online
- Aplicação está rodando

**Passos:**

1. Desligar o banco de dados Oracle
2. Fazer uma chamada ao `/health` (deve retornar unhealthy)
3. Religar o banco de dados Oracle
4. **AGUARDAR 10-15 segundos** para o pool de conexões detectar
5. Fazer nova chamada ao `/health`

**Teste:**

```bash
# Após religar o banco, aguardar e testar
sleep 15
curl -X GET http://localhost:8080/health | jq
```

**Resultado Esperado (1ª tentativa após religar):**
Pode retornar "unhealthy" na primeira tentativa, mas deve mostrar tentativa de recuperação nos logs.

**Resultado Esperado (2ª tentativa):**

```json
{
  "status": "healthy",
  "timestamp": "2026-01-23T18:00:00-03:00",
  "checks": {
    "database": {
      "status": "healthy",
      "message": "Database connection recovered"
    },
    "rabbitmq": {
      "status": "healthy",
      "message": "RabbitMQ connection is active"
    }
  }
}
```

**Código HTTP:** 200 OK

**Logs Esperados:**

```
[WARN] Database connection broken, attempting to reconnect: ...
[INFO] Database connection recovered after retry
```

---

### Cenário 4: Teste Contínuo de Recuperação

**Objetivo:** Validar que o sistema se recupera consistentemente

**Script de Teste:**

```bash
#!/bin/bash
echo "Testando recuperação automática do health check..."
echo "Certifique-se de que o banco foi religado!"
echo ""

for i in {1..10}; do
  echo "Tentativa $i:"
  STATUS=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/health)
  RESPONSE=$(curl -s http://localhost:8080/health | jq -r '.checks.database.status')
  echo "  HTTP: $STATUS | Database: $RESPONSE"

  if [ "$RESPONSE" == "healthy" ]; then
    echo "  ✅ Recuperação bem-sucedida!"
    break
  fi

  sleep 3
done
```

**Resultado Esperado:**

- Primeiras tentativas podem retornar "unhealthy"
- Dentro de 3-5 tentativas (9-15 segundos), deve retornar "healthy"
- Mensagem "✅ Recuperação bem-sucedida!" deve aparecer

---

## Comportamento do Pool de Conexões

### Configuração Atual (oracle.go):

```go
db.SetMaxOpenConns(50)                    // Máximo de 50 conexões abertas
db.SetMaxIdleConns(25)                    // Mantém 25 conexões ociosas
db.SetConnMaxLifetime(30 * time.Minute)   // Renova conexões a cada 30 min
db.SetConnMaxIdleTime(10 * time.Minute)   // Fecha ociosas após 10 min
```

### Como Funciona a Recuperação:

1. **Detecção:** Health check detecta "broken pipe" ou erro similar
2. **Retry:** Tenta um segundo ping com timeout de 3 segundos
3. **Pool:** O pool do Go automaticamente:
   - Descarta a conexão quebrada
   - Cria uma nova conexão ao banco
   - Retorna a nova conexão para o ping
4. **Resultado:** Se o banco estiver online, o segundo ping terá sucesso

### Tempo de Recuperação:

- **Imediato:** Se o pool já tiver conexões válidas
- **3-8 segundos:** Se precisar criar nova conexão
- **Timeout:** Máximo de 8 segundos (5s primeiro ping + 3s retry)

---

## Troubleshooting

### Problema: Não recupera mesmo após religar o banco

**Possíveis Causas:**

1. Pool de conexões ainda não detectou que o banco voltou
2. Firewall bloqueando novas conexões
3. Banco ainda não está totalmente online

**Solução:**

- Aguardar mais tempo (até 30 segundos)
- Verificar logs do banco de dados
- Testar conexão direta: `sqlplus user/pass@host:port/service`

### Problema: Demora muito para recuperar

**Possíveis Causas:**

1. Configuração de timeout muito alta
2. Rede lenta
3. Banco sobrecarregado

**Solução:**

- Verificar configuração de timeouts no oracle.go
- Testar latência de rede: `ping host`
- Verificar carga do banco de dados

---

## Monitoramento em Produção

### Kubernetes Liveness Probe

```yaml
livenessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 30
  periodSeconds: 10
  timeoutSeconds: 5
  failureThreshold: 3
```

### Kubernetes Readiness Probe

```yaml
readinessProbe:
  httpGet:
    path: /health
    port: 8080
  initialDelaySeconds: 10
  periodSeconds: 5
  timeoutSeconds: 5
  failureThreshold: 2
```

### Alertas Recomendados

1. **Alerta Crítico:** Health check unhealthy por mais de 2 minutos
2. **Alerta Warning:** Health check unhealthy por mais de 30 segundos
3. **Alerta Info:** Recuperação automática detectada (logs WARN → INFO)

---

## Conclusão

O health check implementado:

- ✅ Detecta falhas de conexão automaticamente
- ✅ Tenta recuperar usando retry com timeout
- ✅ Aproveita o pool de conexões do Go
- ✅ Fornece logs detalhados para troubleshooting
- ✅ É compatível com orquestradores (Kubernetes, Docker Swarm)
- ✅ Retorna códigos HTTP apropriados (200 OK / 503 Service Unavailable)
