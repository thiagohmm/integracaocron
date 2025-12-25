package entities

type Promotion struct {
	IPM_ID          int    `json:"ipm_id"`
	Json            string `json:"json"`
	DATARECEBIMENTO string `json:"datarecebimento"`
}

type PromotionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
