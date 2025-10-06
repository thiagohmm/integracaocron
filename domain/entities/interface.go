package entities

import "time"

type PromotionRepository interface {
	Dopkg_promotion(pIprId int) (*PromotionResult, error)
	GetIntegrRMSPromocaoIN() ([]Promotion, error)
	DeletePorObjeto(ipmID int) error
}

// ParameterRepository handles system parameters
type ParameterRepository interface {
	ListByCodeParameter(codigo string) (*IParameter, error)
	Update(param *IParameter) error
}

// IntegrationRepository handles integration cleanup operations
type IntegrationRepository interface {
	// Transaction removal methods
	RemoveIntegrationCombo(dataCorte time.Time, expurgo ...string) error
	ClearIntegrationPackagingByCutOffDate(dataCorte time.Time, expurgo ...string) error
	RemoverTransacaoIntegracaoEstruturaMercadologica(dataCorte time.Time, expurgo ...string) error
	RemoverTransacaoIntegracaoProduto(dataCorte time.Time, expurgo ...string) error
	RemoverTransacaoIntegracaoPromocao(dataCorte time.Time, expurgo ...string) error

	// Data movement methods
	MoveIntegrationMarketingStructure(dataCorte time.Time) error
	MoveIntegrationProductStaging(dataCorte time.Time) error
	MoveIntegrationPackagingStaging(dataCorte time.Time) error
	MoveIntegrationComboStaging(dataCorte time.Time) error
	MoveIntegrationPromotionStaging(dataCorte time.Time) error

	// Expiry methods
	GetIntegrationUpdateComboByDate(dataCorte time.Time) ([]IntegrationCombo, error)
	DeleteIntegrationCombo(idIntegracaoCombo int) error
	UpdateExpiredSlaSolicitation() error
}

// NetworkRepository handles network operations
type NetworkRepository interface {
	GetNetwork() ([]Network, error)
	ListByAllByIdDealerNew(idRevendedor int) ([]DealerNetwork, error)
	ReplicateProductNetwork(idRede int) error
	GetNetworkReplicadosByDealer(idRevendedor int) ([]interface{}, error)
	GetProductsByReplicateNetworkServiceNew(idRevendedor int) ([]ProductSelect, error)
}
