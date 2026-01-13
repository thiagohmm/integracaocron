package entities

import "time"

type Promotion struct {
	IPM_ID          int    `json:"ipm_id"`
	Json            string `json:"json"`
	DATARECEBIMENTO string `json:"datarecebimento"`
}

type PromotionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// PromotionIntegrationStaging represents the promotion integration staging entity
type PromotionIntegrationStaging struct {
	IdIntegracaoPromocaoStaging int       `json:"id_integracao_promocao_staging" db:"ID_INTEGRACAO_PROMOCAO_STAGING"`
	IdPromocao                  int       `json:"id_promocao" db:"ID_PROMOCAO"`
	IdRevendedor                int       `json:"id_revendedor" db:"ID_REVENDEDOR"`
	Json                        string    `json:"json" db:"JSON"`
	DataAtualizacao             time.Time `json:"data_atualizacao" db:"DATA_ATUALIZACAO"`
}
