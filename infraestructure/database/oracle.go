package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"time"

	config "github.com/thiagohmm/integracaocron/configuration"

	go_ora "github.com/sijms/go-ora/v2"
)

func ConectarBanco(cfg *config.Conf) (*sql.DB, error) {
	// Configurar as opções de URL com timeouts mais generosos e configurações compatíveis com Oracle Cloud
	urlOptions := map[string]string{
		"CONNECTION TIMEOUT": "90",    // Timeout de 90 segundos para Oracle Cloud
		"TIMEOUT":            "90",    // Timeout geral
		"RETRY COUNT":        "20",    // Número de tentativas de reconexão
		"RETRY DELAY":        "3",     // Delay entre tentativas (segundos)
		"ssl":                "true",  // Habilitar SSL para tcps
		"ssl verify":         "false", // Desabilitar verificação de certificado SSL
		// "wallet":           "./wallet", // descomente se estiver usando wallet
	}

	// Construir a string de conexão
	connStr := go_ora.BuildUrl(cfg.Host, cfg.Port, cfg.ServiceName, cfg.DBUser, cfg.DBPassword, urlOptions)

	log.Printf("Tentando conectar ao Oracle Cloud: %s:%d/%s", cfg.Host, cfg.Port, cfg.ServiceName)

	var db *sql.DB
	var err error
	maxRetries := 3

	// Tenta reconectar com número máximo de tentativas
	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("Tentativa de conexão %d/%d...", attempt, maxRetries)
		
		db, err = sql.Open(cfg.DBDriver, connStr)
		if err != nil {
			log.Printf("Erro ao abrir a conexão (tentativa %d/%d): %v", attempt, maxRetries, err)
			if attempt < maxRetries {
				time.Sleep(5 * time.Second)
				continue
			}
			return nil, fmt.Errorf("falha ao abrir conexão após %d tentativas: %w", maxRetries, err)
		}

		// Usa um contexto com timeout maior para a operação de ping
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		err = db.PingContext(ctx)
		cancel()

		if err != nil {
			log.Printf("Erro ao verificar a conexão (tentativa %d/%d): %v", attempt, maxRetries, err)
			// IMPORTANTE: Fechar a conexão que falhou antes de tentar novamente
			if closeErr := db.Close(); closeErr != nil {
				log.Printf("Erro ao fechar conexão anterior: %v", closeErr)
			}
			if attempt < maxRetries {
				time.Sleep(5 * time.Second)
				continue
			}
			return nil, fmt.Errorf("falha ao pingar banco após %d tentativas: %w", maxRetries, err)
		}

		log.Println("✓ Conexão estabelecida com sucesso com o banco Oracle")
		break
	}

	// Configurar pool de conexões para Oracle Cloud
	// Oracle Cloud Autonomous Database geralmente tem limites de conexão mais baixos
	db.SetMaxOpenConns(10)  // Reduzido de 25 para 10 (melhor para Oracle Cloud)
	db.SetMaxIdleConns(5)   // Reduzido de 10 para 5 (libera recursos mais rápido)
	
	// Configurações de lifetime mais agressivas para Oracle Cloud
	// Oracle Cloud pode fechar conexões inativas após 30 minutos
	db.SetConnMaxLifetime(10 * time.Minute)  // Força renovação a cada 10 minutos
	db.SetConnMaxIdleTime(5 * time.Minute)   // Fecha conexões ociosas após 5 minutos

	log.Println("✓ Pool de conexões configurado: MaxOpen=10, MaxIdle=5, MaxLifetime=10m, MaxIdleTime=5m")

	// Validar a conexão com uma query simples
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	var result int
	if err := db.QueryRowContext(ctx, "SELECT 1 FROM DUAL").Scan(&result); err != nil {
		log.Printf("Aviso: Falha na query de validação: %v", err)
	} else {
		log.Println("✓ Validação da conexão bem-sucedida")
	}

	return db, nil
}
