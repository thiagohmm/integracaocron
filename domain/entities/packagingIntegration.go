package entities

import "time"

// PackagingIntegrationStaging represents the packaging integration staging entity
type PackagingIntegrationStaging struct {
	IdIntegracaoEmbalagemStaging int       `json:"id_integracao_embalagem_staging" db:"IDINTEGRACAOEMBALAGEMSTAGING"`
	IdEmbalagemProduto           int       `json:"id_embalagem_produto" db:"IDEMBALAGEMPRODUTO"`
	IdRevendedor                 int       `json:"id_revendedor" db:"IDREVENDEDOR"`
	Json                         string    `json:"json" db:"JSON"`
	DataAtualizacao              time.Time `json:"data_atualizacao" db:"DATAATUALIZACAO"`
}

// RetPackaging represents the return structure for packaging integration
type RetPackaging struct {
	IdEmbalagemProduto int `json:"id_embalagem_produto"`
	Mix                Mix `json:"mix"`
}

// Mix represents the packaging mix structure
type Mix struct {
	CodMix  string         `json:"cod_mix"`
	DescMix string         `json:"desc_mix"`
	DataDe  time.Time      `json:"data_de"`
	DataAte time.Time      `json:"data_ate"`
	Status  string         `json:"status"`
	Grupos  []GroupItemMix `json:"grupos"`
	DtAlt   time.Time      `json:"dt_alt"`
}

// GroupItemMix represents a group of items in a packaging mix
type GroupItemMix struct {
	Desc     string         `json:"desc"`
	QtdeItem int            `json:"qtde_item"`
	Items    []ItemGroupMix `json:"items"`
}

// ItemGroupMix represents an individual item in a group mix
type ItemGroupMix struct {
	CodItem  string `json:"cod_item"`
	CodBarra string `json:"cod_barra"`
}
