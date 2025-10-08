package entities

import "time"

// IParameter represents a system parameter
type IParameter struct {
	IdParametro int    `json:"id_parametro" db:"ID_PARAMETRO"`
	Codigo      string `json:"codigo" db:"CODIGO"`
	Valor       string `json:"valor" db:"VALOR"`
	Ambiente    string `json:"ambiente" db:"AMBIENTE"`
	Descricao   string `json:"descricao" db:"DESCRICAO"`
}

// IFilterParameter represents filter criteria for parameters
type IFilterParameter struct {
	Codigo   string `json:"codigo"`
	Ambiente string `json:"ambiente"`
}

// IntegrationCombo represents combo integration data
type IntegrationCombo struct {
	IdIntegracaoCombo int        `json:"id_integracao_combo" db:"ID_INTEGRACAO_COMBO"`
	IdRevendedor      int        `json:"id_revendedor" db:"ID_REVENDEDOR"`
	IdComboPromocao   int        `json:"id_combo_promocao" db:"ID_COMBO_PROMOCAO"`
	Enviando          *string    `json:"enviando" db:"ENVIANDO"`
	Json              *string    `json:"json" db:"JSON"`
	DataAtualizacao   *time.Time `json:"data_atualizacao" db:"DATA_ATUALIZACAO"`
	Transacao         *string    `json:"transacao" db:"TRANSACAO"`
	DataInicioEnvio   *time.Time `json:"data_inicio_envio" db:"DATA_INICIO_ENVIO"`
}

// IntegrationPackaging represents packaging integration data
type IntegrationPackaging struct {
	IdIntegracaoEmbalagem int        `json:"id_integracao_embalagem" db:"ID_INTEGRACAO_EMBALAGEM"`
	IdRevendedor          int        `json:"id_revendedor" db:"ID_REVENDEDOR"`
	IdEmbalagemProduto    int        `json:"id_embalagem_produto" db:"ID_EMBALAGEM_PRODUTO"`
	Enviando              *string    `json:"enviando" db:"ENVIANDO"`
	Json                  *string    `json:"json" db:"JSON"`
	DataAtualizacao       *time.Time `json:"data_atualizacao" db:"DATA_ATUALIZACAO"`
	Transacao             *string    `json:"transacao" db:"TRANSACAO"`
	DataInicioEnvio       *time.Time `json:"data_inicio_envio" db:"DATA_INICIO_ENVIO"`
}

// IntegrationMarketingStructure represents marketing structure integration data
type IntegrationMarketingStructure struct {
	IdIntegracaoEstruturaMercadologica int        `json:"id_integracao_estrutura_mercadologica" db:"ID_INTEGRACAO_ESTRUTURA_MERCADOLOGICA"`
	IdRevendedor                       int        `json:"id_revendedor" db:"ID_REVENDEDOR"`
	IdEstruturaMercadologica           int        `json:"id_estrutura_mercadologica" db:"ID_ESTRUTURA_MERCADOLOGICA"`
	Enviando                           *string    `json:"enviando" db:"ENVIANDO"`
	Json                               *string    `json:"json" db:"JSON"`
	DataAtualizacao                    *time.Time `json:"data_atualizacao" db:"DATA_ATUALIZACAO"`
	Transacao                          *string    `json:"transacao" db:"TRANSACAO"`
	DataInicioEnvio                    *time.Time `json:"data_inicio_envio" db:"DATA_INICIO_ENVIO"`
}

// IntegrationProduct represents product integration data
type IntegrationProduct struct {
	ID             int       `json:"id" db:"ID"`
	DataIntegracao time.Time `json:"data_integracao" db:"DATA_INTEGRACAO"`
}

// IntegrationPromotion represents promotion integration data
type IntegrationPromotion struct {
	ID             int       `json:"id" db:"ID"`
	DataIntegracao time.Time `json:"data_integracao" db:"DATA_INTEGRACAO"`
}

// Network represents a network entity
type Network struct {
	IdRede                 int        `json:"id_rede" db:"ID_REDE"`
	DescricaoRede          string     `json:"descricao_rede" db:"DESCRICAO_REDE"`
	IdRevendedor           int        `json:"id_revendedor" db:"ID_REVENDEDOR"`
	StatusRede             *string    `json:"status_rede" db:"STATUS_REDE"`
	ReplicarProduto        *string    `json:"replicar_produto" db:"REPLICAR_PRODUTO"`
	DataCadastro           *time.Time `json:"data_cadastro" db:"DATA_CADASTRO"`
	DataAtualizacao        *time.Time `json:"data_atualizacao" db:"DATA_ATUALIZACAO"`
	PermiteReplicarProduto *string    `json:"permite_replicar_produto" db:"PERMITE_REPLICAR_PRODUTO"`
	UsuarioReplicou        string     `json:"usuario_replicou" db:"USUARIO_REPLICOU"`
}

// ProductReplicate represents a product replication entity
type ProductReplicate struct {
	IdRevendedor int `json:"id_revendedor" db:"ID_REVENDEDOR"`
	IdProduto    int `json:"id_produto" db:"ID_PRODUTO"`
}

// Success represents a success response
type Success struct {
	Message string `json:"message"`
	Success bool   `json:"success"`
}

// DealerNetwork represents a dealer network entity
type DealerNetwork struct {
	IdRevendedor int `json:"id_revendedor" db:"ID_REVENDEDOR"`
}

// ProductSelect represents a product for selection/replication
type ProductSelect struct {
	Cod string `json:"cod" db:"COD"`
}
