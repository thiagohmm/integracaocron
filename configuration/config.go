package configuration

import (
	"fmt"
	"log"
	"regexp"
	"strconv"

	"github.com/spf13/viper"
)

type Conf struct {
	DBDriver           string `mapstructure:"DB_DIALECT"`
	DBUser             string `mapstructure:"DB_USER"`
	DBPassword         string `mapstructure:"DB_PASSWD"`
	DBSchema           string `mapstructure:"DB_SCHEMA"`
	DBConnect          string `mapstructure:"DB_CONNECTSTRING"`
	ServiceName        string
	Port               int
	Host               string
	ENV_RABBITMQ       string `mapstructure:"ENV_RABBITMQ"`
	ENV_REDIS_ADDR     string `mapstructure:"ENV_REDIS_ADDRESS"`
	ENV_REDIS_PASSWORD string `mapstructure:"ENV_REDIS_PASSWORD"`
	ENV_REDIS_EXPIRE   int    `mapstructure:"ENV_REDIS_EXPIRE"`
}

type Dados struct {
	ServiceName string
	Port        int
	Host        string
}

func extrairDados(descricao string) (Dados, error) {
	var dados Dados

	// Expressões regulares para extrair os valores
	serviceNamePattern := regexp.MustCompile(`service_name=([\w.]+)`)
	portPattern := regexp.MustCompile(`port=(\d+)`)
	hostPattern := regexp.MustCompile(`host=([\d.]+)`)

	// Encontrar os valores usando as expressões regulares
	serviceNameMatch := serviceNamePattern.FindStringSubmatch(descricao)
	portMatch := portPattern.FindStringSubmatch(descricao)
	hostMatch := hostPattern.FindStringSubmatch(descricao)

	// Verificar se todos os valores foram encontrados
	if len(serviceNameMatch) < 2 || len(portMatch) < 2 || len(hostMatch) < 2 {
		return dados, fmt.Errorf("não foi possível extrair todos os dados")
	}

	dados.ServiceName = serviceNameMatch[1]
	dados.Port, _ = strconv.Atoi(portMatch[1])
	dados.Host = hostMatch[1]

	return dados, nil
}

// Carrega as configurações do arquivo .env e das variáveis de ambiente
func LoadConfig(path string) (*Conf, error) {
	var cfg Conf
	viper.SetConfigName("app_config")
	viper.SetConfigType("env")
	viper.AddConfigPath(path)
	viper.SetConfigFile(".env")
	viper.AutomaticEnv()

	err := viper.ReadInConfig()
	if err != nil {
		fmt.Println("Não foi possível ler o arquivo .env, tentando variáveis de ambiente")
		cfg.DBDriver = viper.GetString("DB_DIALECT")
		cfg.DBUser = viper.GetString("DB_USER")
		cfg.DBPassword = viper.GetString("DB_PASSWD")
		cfg.DBSchema = viper.GetString("DB_SCHEMA")
		cfg.DBConnect = viper.GetString("DB_CONNECTSTRING")
		cfg.ENV_RABBITMQ = viper.GetString("ENV_RABBITMQ")

		cfg.ENV_REDIS_ADDR = viper.GetString("ENV_REDIS_ADDRESS")
		cfg.ENV_REDIS_PASSWORD = viper.GetString("ENV_REDIS_PASSWORD")
		cfg.ENV_REDIS_EXPIRE = viper.GetInt("ENV_REDIS_EXPIRE")
	} else {
		err = viper.Unmarshal(&cfg)
		if err != nil {
			//panic(err)
			log.Printf("Erro ao carregar configurações: %v", err)
		}
	}

	// Extrair os dados da string de conexão
	dados, err := extrairDados(cfg.DBConnect)
	if err != nil {
		return &cfg, err
	}

	// Preencher os campos na struct de configuração
	cfg.ServiceName = dados.ServiceName
	cfg.Port = dados.Port
	cfg.Host = dados.Host

	return &cfg, nil
}
