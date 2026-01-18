package database

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	config "github.com/thiagohmm/integracaocron/configuration"

	go_ora "github.com/sijms/go-ora/v2"
)

func ConectarBanco(cfg *config.Conf) (*sql.DB, error) {
	// Validar configuração antes de tentar conectar
	if cfg.Host == "" {
		return nil, fmt.Errorf("host não configurado (DB_CONNECTSTRING ou host extraído está vazio)")
	}
	if cfg.Port == 0 {
		return nil, fmt.Errorf("porta não configurada (DB_CONNECTSTRING ou port extraído está vazio)")
	}
	if cfg.ServiceName == "" {
		return nil, fmt.Errorf("service name não configurado (DB_CONNECTSTRING ou service_name extraído está vazio)")
	}
	if cfg.DBUser == "" {
		return nil, fmt.Errorf("usuário do banco não configurado (DB_USER está vazio)")
	}
	if cfg.DBPassword == "" {
		return nil, fmt.Errorf("senha do banco não configurada (DB_PASSWD está vazio)")
	}

	log.Printf("📋 Configuração de conexão:")
	log.Printf("   Host: %s", cfg.Host)
	log.Printf("   Port: %d", cfg.Port)
	log.Printf("   Service Name: %s", cfg.ServiceName)
	log.Printf("   User: %s", cfg.DBUser)
	log.Printf("   Driver: %s", cfg.DBDriver)

	// Para Oracle Cloud Autonomous Database, pode ser necessário usar o hostname completo
	// Se o host for um IP privado e não funcionar, tente usar o hostname do service name
	hostToUse := cfg.Host
	
	// Se o host é um IP privado (10.x.x.x, 172.16-31.x.x, 192.168.x.x), 
	// tentar usar o hostname do service name se disponível
	if isPrivateIP(cfg.Host) && cfg.ServiceName != "" {
		// O service name do Oracle Cloud geralmente contém o hostname
		// Exemplo: gbe8f3f2dbbc562_dwpdbprd_low.adb.oraclecloud.com
		if containsHostname(cfg.ServiceName) {
			log.Printf("⚠️  Host é um IP privado (%s), mas o service name contém hostname", cfg.Host)
			log.Printf("💡 Dica: Para Oracle Cloud, use o hostname completo no DB_CONNECTSTRING")
			log.Printf("   Exemplo: host=gbe8f3f2dbbc562_dwpdbprd_low.adb.oraclecloud.com")
		}
	}

	// Configurar as opções de URL compatíveis com go-ora v2
	// Documentação: https://github.com/sijms/go-ora
	urlOptions := map[string]string{
		"TIMEOUT":     "90",    // Timeout geral em segundos
		"ssl":         "true",  // Habilitar SSL para Oracle Cloud
		"ssl verify":  "false", // Desabilitar verificação de certificado SSL
		// "wallet":    "./wallet", // descomente se estiver usando wallet
	}

	// Construir a string de conexão
	connStr := go_ora.BuildUrl(hostToUse, cfg.Port, cfg.ServiceName, cfg.DBUser, cfg.DBPassword, urlOptions)

	log.Printf("🔌 Tentando conectar ao Oracle Cloud: %s:%d/%s", hostToUse, cfg.Port, cfg.ServiceName)
	log.Printf("   String de conexão (mascarada): %s:%d/%s", hostToUse, cfg.Port, cfg.ServiceName)

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
			log.Printf("❌ Erro ao verificar a conexão (tentativa %d/%d): %v", attempt, maxRetries, err)
			
			// Diagnóstico adicional para erros comuns
			errStr := err.Error()
			if strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "connect: connection refused") {
				log.Printf("⚠️  DIAGNÓSTICO: Connection refused - possíveis causas:")
				log.Printf("   1. IP/Hostname incorreto ou não acessível")
				log.Printf("   2. Porta bloqueada por firewall")
				log.Printf("   3. Para Oracle Cloud Autonomous Database, use o hostname completo")
				log.Printf("      Exemplo: host=gbe8f3f2dbbc562_dwpdbprd_low.adb.oraclecloud.com")
				log.Printf("   4. Verifique se está na mesma rede/VPN se for IP privado")
				log.Printf("   5. Verifique se o Oracle Cloud permite conexões do seu IP")
				log.Printf("   Host atual: %s:%d", cfg.Host, cfg.Port)
				if isPrivateIP(cfg.Host) {
					log.Printf("   ⚠️  IP privado detectado (%s) - pode não ser acessível do ambiente atual", cfg.Host)
					log.Printf("   💡 Solução: Use o hostname completo do Oracle Cloud no DB_CONNECTSTRING")
				}
			} else if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "i/o timeout") {
				log.Printf("⚠️  DIAGNÓSTICO: Timeout na conexão - possíveis causas:")
				log.Printf("   1. Firewall bloqueando a conexão")
				log.Printf("   2. Rede lenta ou instável")
				log.Printf("   3. Oracle Cloud não está respondendo")
			} else if strings.Contains(errStr, "no such host") || strings.Contains(errStr, "unknown host") {
				log.Printf("⚠️  DIAGNÓSTICO: Host não encontrado - verifique o hostname/IP")
			}
			
			// IMPORTANTE: Fechar a conexão que falhou antes de tentar novamente
			if closeErr := db.Close(); closeErr != nil {
				log.Printf("Erro ao fechar conexão anterior: %v", closeErr)
			}
			if attempt < maxRetries {
				log.Printf("⏳ Aguardando 5 segundos antes da próxima tentativa...")
				time.Sleep(5 * time.Second)
				continue
			}
			
			// Mensagem de erro mais detalhada
			return nil, fmt.Errorf("falha ao conectar ao banco após %d tentativas\n"+
				"Host: %s:%d\n"+
				"Service: %s\n"+
				"Erro: %w\n"+
				"Dica: Verifique DB_CONNECTSTRING no .env ou variáveis de ambiente", 
				maxRetries, cfg.Host, cfg.Port, cfg.ServiceName, err)
		}

		log.Println("✓ Conexão estabelecida com sucesso com o banco Oracle")
		break
	}

	// Configurar pool de conexões otimizado para performance
	// Aumentado para melhor throughput com múltiplos workers
	db.SetMaxOpenConns(50)  // Aumentado para suportar mais workers concorrentes
	db.SetMaxIdleConns(25)  // Mantém mais conexões ociosas para reutilização rápida
	
	// Configurações de lifetime otimizadas
	db.SetConnMaxLifetime(30 * time.Minute)  // Renovação a cada 30 minutos
	db.SetConnMaxIdleTime(10 * time.Minute)  // Fecha conexões ociosas após 10 minutos

	log.Println("✓ Pool de conexões configurado: MaxOpen=50, MaxIdle=25, MaxLifetime=30m, MaxIdleTime=10m")

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

// isPrivateIP verifica se um IP é privado
func isPrivateIP(ip string) bool {
	// Verifica padrões de IP privado
	// 10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16
	return len(ip) >= 7 && (
		ip[:3] == "10." ||
		(len(ip) >= 8 && ip[:4] == "172.") ||
		(len(ip) >= 8 && ip[:4] == "192."))
}

// containsHostname verifica se uma string contém um hostname (contém pontos e não é apenas números)
func containsHostname(s string) bool {
	// Verifica se contém pontos (indicando hostname) e não é apenas um IP
	return len(s) > 0 && (strings.Contains(s, ".") && !isPrivateIP(s))
}
