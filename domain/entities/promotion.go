package entities

type Promotion struct {
	IPMD_ID         int    `json:"ipmd_id"`
	Json            string `json:"json"`
	DATARECEBIMENTO string `json:"datarecebimento"`
}

type PromotionResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}
