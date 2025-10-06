package entities

import "time"

// IParameter represents a system parameter
type IParameter struct {
	Codigo   string `json:"codigo" db:"CODIGO"`
	Valor    string `json:"valor" db:"VALOR"`
	Ambiente string `json:"ambiente" db:"AMBIENTE"`
}

// IntegrationCombo represents combo integration data
type IntegrationCombo struct {
	IdIntegracaoCombo int       `json:"id_integracao_combo" db:"ID_INTEGRACAO_COMBO"`
	DataIntegracao    time.Time `json:"data_integracao" db:"DATA_INTEGRACAO"`
}

// IntegrationPackaging represents packaging integration data
type IntegrationPackaging struct {
	ID             int       `json:"id" db:"ID"`
	DataIntegracao time.Time `json:"data_integracao" db:"DATA_INTEGRACAO"`
}

// IntegrationMarketingStructure represents marketing structure integration data
type IntegrationMarketingStructure struct {
	ID             int       `json:"id" db:"ID"`
	DataIntegracao time.Time `json:"data_integracao" db:"DATA_INTEGRACAO"`
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
	IdRede       int `json:"id_rede" db:"ID_REDE"`
	IdRevendedor int `json:"id_revendedor" db:"ID_REVENDEDOR"`
}

// DealerNetwork represents a dealer network entity
type DealerNetwork struct {
	IdRevendedor int `json:"id_revendedor" db:"ID_REVENDEDOR"`
}

// ProductSelect represents a product for selection/replication
type ProductSelect struct {
	Cod string `json:"cod" db:"COD"`
}
