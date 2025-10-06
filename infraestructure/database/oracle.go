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
	// Configurar as opções de URL
	urlOptions := map[string]string{
		"CONNECTION TIMEOUT": "10",
		"ssl":                "true",  // ou habilite o SSL
		"ssl verify":         "false", // desabilita a verificação de certificado SSL
		// "wallet":           "./wallet", // descomente se estiver usando wallet
	}

	// Construir a string de conexão
	connStr := go_ora.BuildUrl(cfg.Host, cfg.Port, cfg.ServiceName, cfg.DBUser, cfg.DBPassword, urlOptions)

	var db *sql.DB
	var err error

	// Tenta reconectar automaticamente até conseguir
	for {
		db, err = sql.Open(cfg.DBDriver, connStr)
		if err != nil {
			log.Printf("Erro ao abrir a conexão: %v. Tentando novamente...", err)
			time.Sleep(2 * time.Second)
			continue
		}

		// Usa um contexto com timeout para a operação de ping
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err = db.PingContext(ctx)
		cancel()

		if err != nil {
			log.Printf("Erro ao verificar a conexão: %v. Tentando novamente...", err)
			time.Sleep(2 * time.Second)
			continue
		}

		log.Println("Conexão estabelecida com sucesso com o banco Oracle")
		break
	}

	// Retorna a conexão ou um erro (nunca chegará aqui com erro se a conexão foi eventualmente bem-sucedida)
	if db == nil {
		return nil, fmt.Errorf("não foi possível conectar ao banco Oracle")
	}
	return db, nil
}
