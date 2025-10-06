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

## 🔍 Monitoramento

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
  "tabela": "LogIntegrRMS",
  "fields": ["TRANSACAO", "TABELA", "DATARECEBIMENTO", "DATAPROCESSAMENTO", "STATUSPROCESSAMENTO", "JSON", "DESCRICAOERRO"],
  "values": ["IN", "PROMOCAO", "2025-10-06 12:00:00", "2025-10-06 12:05:00", 0, "{...}", "Processamento realizado com sucesso."]
}
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

## 📄 Licença

Este projeto é propriedade privada da empresa.

## 👥 Contribuição

Para contribuir com o projeto, siga o padrão de commits convencionais e abra um Pull Request.

---

**Desenvolvido com ❤️ em Go**