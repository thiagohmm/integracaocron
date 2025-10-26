# Formatos de Mensagem RabbitMQ - Guia de Uso

Este documento descreve os formatos de mensagem aceitos pelo listener do RabbitMQ para integração de produtos e promoções.

## Formatos Suportados

O sistema aceita **3 formatos diferentes** de mensagem para flexibilidade máxima:

### 1. Formato Simples (Recomendado para Novos Sistemas)

Envie apenas o tipo de integração como string:

```json
"promocao"
```

ou

```json
"produto"
```

ou

```json
"promocao_normalizacao"
```

### 2. Formato com `type_message`

```json
{
  "type_message": "promocao"
}
```

ou

```json
{
  "type_message": "produto",
  "dados": {
    "algum_parametro": "valor"
  }
}
```

### 3. Formato Legado com `tipoIntegracao`

```json
{
  "tipoIntegracao": "Promocao",
  "dados": {}
}
```

## Tipos de Integração Aceitos

### 1. Promoção

**Valores aceitos (case-insensitive):**
- `"promocao"`
- `"Promocao"`

**Exemplos de uso:**

```json
"promocao"
```

```json
{
  "type_message": "promocao"
}
```

```json
{
  "tipoIntegracao": "Promocao",
  "dados": {}
}
```

**O que faz:**
- Processa promoções do sistema
- Executa integração com revendedores
- Atualiza dados de promoções

### 2. Produto

**Valores aceitos (case-insensitive):**
- `"produto"`
- `"Produto"`

**Exemplos de uso:**

```json
"produto"
```

```json
{
  "type_message": "produto"
}
```

```json
{
  "tipoIntegracao": "Produto",
  "dados": {}
}
```

**O que faz:**
- Importa produtos da integração RMS
- Processa dados da tabela `INTEGR_RMS_PRODUTO_IN`
- Executa procedure Oracle `pkg_integra_produto.prc_integra_hermes`
- Valida e salva produtos no sistema

### 3. Normalização de Promoção

**Valores aceitos (case-insensitive):**
- `"promocao_normalizacao"`
- `"PromocaoNormalizacao"`

**Exemplos de uso:**

```json
"promocao_normalizacao"
```

```json
{
  "type_message": "promocao_normalizacao"
}
```

```json
{
  "tipoIntegracao": "PromocaoNormalizacao",
  "dados": {}
}
```

**O que faz:**
- Normaliza dados de promoções
- Remove itens duplicados dos grupos de promoção
- Atualiza contadores de itens (`qtdeItem`)
- Processa todos os registros da tabela `INTEGRACAO_PROMOCAO`

## Detecção Automática de Formato

O listener detecta automaticamente qual formato está sendo usado:

1. **Primeiro**, tenta ler `type_message` do JSON
2. **Segundo**, tenta ler `tipoIntegracao` (formato legado)
3. **Terceiro**, tenta interpretar a mensagem como string simples

## Exemplos de Publicação

### Usando RabbitMQ Admin CLI

```bash
# Formato simples
rabbitmqadmin publish routing_key=integracaoCron payload='"promocao"'

rabbitmqadmin publish routing_key=integracaoCron payload='"produto"'

# Formato com type_message
rabbitmqadmin publish routing_key=integracaoCron \
  payload='{"type_message":"promocao"}'

rabbitmqadmin publish routing_key=integracaoCron \
  payload='{"type_message":"produto"}'
```

### Usando Go (amqp library)

```go
// Formato simples
ch.Publish("", "integracaoCron", false, false,
    amqp.Publishing{
        ContentType: "application/json",
        Body:        []byte(`"promocao"`),
    })

// Formato com type_message
ch.Publish("", "integracaoCron", false, false,
    amqp.Publishing{
        ContentType: "application/json",
        Body:        []byte(`{"type_message":"produto"}`),
    })
```

### Usando Node.js/TypeScript

```typescript
// Formato simples
channel.sendToQueue('integracaoCron', 
    Buffer.from(JSON.stringify("promocao")));

// Formato com type_message
channel.sendToQueue('integracaoCron', 
    Buffer.from(JSON.stringify({
        type_message: "produto"
    })));
```

### Usando Python (pika)

```python
import json
import pika

connection = pika.BlockingConnection(
    pika.ConnectionParameters('localhost'))
channel = connection.channel()

# Formato simples
channel.basic_publish(
    exchange='',
    routing_key='integracaoCron',
    body=json.dumps("promocao"))

# Formato com type_message
channel.basic_publish(
    exchange='',
    routing_key='integracaoCron',
    body=json.dumps({"type_message": "produto"}))

connection.close()
```

## Logs de Processamento

O sistema registra logs detalhados para cada mensagem:

```
Iniciando processamento de mensagem...
Tipo de integração detectado: promocao
Iniciando processamento de promoção
...
Processamento de promoção concluído
```

## Tratamento de Erros

### Mensagem Inválida

Se a mensagem não contém nem `type_message`, nem `tipoIntegracao`, e não é uma string simples:

```
Campo 'type_message' ou 'tipoIntegracao' inválido ou ausente na mensagem
```

### Tipo Desconhecido

Se o tipo de integração não é reconhecido:

```
Tipo de processo desconhecido: tipo_invalido
```

### Serviço Não Inicializado

Se o use case necessário não foi inicializado:

```
ProductIntegrationUC não foi inicializado
```

## Compatibilidade

✅ **Compatível com:**
- Sistemas legados usando `tipoIntegracao`
- Novos sistemas usando `type_message`
- Mensagens simples de string
- Case-insensitive (`promocao` = `Promocao`)

✅ **Recomendações:**
- Para novos sistemas: use formato simples (`"promocao"`) ou `type_message`
- Para migração gradual: ambos os formatos funcionam simultaneamente
- Para padronização: escolha um formato e documente para sua equipe

## Resumo de Valores Aceitos

| Tipo | Valores Aceitos | Ação |
|------|----------------|------|
| Promoção | `promocao`, `Promocao` | Processa promoções |
| Produto | `produto`, `Produto` | Importa produtos RMS |
| Normalização | `promocao_normalizacao`, `PromocaoNormalizacao` | Normaliza promoções |

## Próximos Passos

1. **Escolha o formato** que melhor se adapta ao seu sistema
2. **Teste** com mensagens de exemplo
3. **Monitore** os logs para confirmar processamento
4. **Padronize** o formato em sua aplicação

---

**Última atualização:** 25 de Outubro de 2025  
**Versão:** 2.0 (Suporte a múltiplos formatos)