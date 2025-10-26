package entities

import "time"

// PromotionNormalization represents the main promotion normalization structure
type PromotionNormalization struct {
	IdIntegracaoPromocao *int       `json:"id_integracao_promocao" db:"ID_INTEGRACAO_PROMOCAO"`
	IdRevendedor         *int       `json:"id_revendedor" db:"ID_REVENDEDOR"`
	IdPromocao           *int       `json:"id_promocao" db:"ID_PROMOCAO"`
	JSON                 string     `json:"json" db:"JSON"`
	DataAtualizacao      *time.Time `json:"data_atualizacao" db:"DATA_ATUALIZACAO"`
	DataRecebimento      *time.Time `json:"data_recebimento" db:"DATA_RECEBIMENTO"`
	Enviando             *string    `json:"enviando" db:"ENVIANDO"`
	Transacao            *string    `json:"transacao" db:"TRANSACAO"`
	DataInicioEnvio      *time.Time `json:"data_inicio_envio" db:"DATA_INICIO_ENVIO"`
}

// PromotionJsonData represents the structure of the JSON field in promotions
type PromotionJsonData struct {
	CodMix string           `json:"codMix"`
	Grupos []PromotionGroup `json:"grupos"`
}

// PromotionGroup represents a group within a promotion
type PromotionGroup struct {
	Desc     string               `json:"desc"`
	Items    []PromotionGroupItem `json:"items"`
	QtdeItem int                  `json:"qtdeItem"`
}

// PromotionGroupItem represents an item within a promotion group
type PromotionGroupItem struct {
	CodBarra string  `json:"codBarra"`
	Desc     string  `json:"desc"`
	Preco    float64 `json:"preco"`
	Qtde     int     `json:"qtde"`
}

// PromotionNormalizationResult represents the result of normalization process
type PromotionNormalizationResult struct {
	Success                bool   `json:"success"`
	Message                string `json:"message"`
	ProcessedCount         int    `json:"processed_count"`
	UpdatedCount           int    `json:"updated_count"`
	TotalRemovedDuplicates int    `json:"total_removed_duplicates"`
}

// PromotionNormalizationLog represents log information for normalization
type PromotionNormalizationLog struct {
	IdIntegracaoPromocao int    `json:"id_integracao_promocao"`
	IdPromocao           int    `json:"id_promocao"`
	IdRevendedor         int    `json:"id_revendedor"`
	CodMix               string `json:"cod_mix"`
	RemovedDuplicates    int    `json:"removed_duplicates"`
}

// Constants for promotion normalization
const (
	MSG_START_IMPORT_PROMOTION_RMS = "Iniciando importação de promoções RMS"
	MSG_END_IMPORT_PROMOTION_RMS   = "Finalizando importação de promoções RMS"

	FRANQUIA       = "FRANQUIA"
	LICENCA        = "LICENCA"
	OXXO_PROPRIA   = "OXXO_PROPRIA"
	SELECT_PROPRIA = "SELECT_PROPRIA"
)
