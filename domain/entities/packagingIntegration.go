package entities

import "time"

// PackagingIntegrationStaging represents the packaging integration staging entity
type PackagingIntegrationStaging struct {
	IdIntegracaoEmbalagemStaging int       `json:"id_integracao_embalagem_staging" db:"ID_INTEGRACAO_EMBALAGEM_STAGING"`
	IdEmbalagemProduto           int       `json:"id_embalagem_produto" db:"ID_EMBALAGEM_PRODUTO"`
	IdRevendedor                 int       `json:"id_revendedor" db:"ID_REVENDEDOR"`
	Json                         string    `json:"json" db:"JSON"`
	DataAtualizacao              time.Time `json:"data_atualizacao" db:"DATA_ATUALIZACAO"`
}
