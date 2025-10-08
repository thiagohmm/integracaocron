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
	Delete(idParametro int) error
	ListById(idParametro int) (*IParameter, error)
	ListGridPerFilter(filter *IFilterParameter) ([]IParameter, error)
	Create(param *IParameter) (*IParameter, error)
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
	GetProductsByReplicateNetworkReplicate(idProduto int) ([]ProductSelect, error)
	GetNetworkByDealer(idDealer int) (*Network, error)
	UpdateNetwork(network *Network) error
	GetNetworkReplicados() ([]ProductReplicate, error)
	ReplicateProductNetworkSP(idNetwork int) error
	RequestReplicateProducts(idNetwork int, userLogin string) (*Success, error)
	MoveIntegrationMarketingStructure(dataCorte time.Time) error
}

// IntegrationMarketingStructureRepository handles marketing structure integration operations
type IntegrationMarketingStructureRepository interface {
	GetIntegrationUpdateByDate(date time.Time) ([]IntegrationMarketingStructure, error)
	GetIntegrations(ip *IntegrationMarketingStructure) ([]IntegrationMarketingStructure, error)
	GetIMSByDate(date time.Time) ([]IntegrationMarketingStructure, error)
	RemoveById(id int) error
	RemoverTransacaoIntegracaoEstruturaMercadologica(dataCorte time.Time, fazExpurgo string) error
}

// IntegrationPackagingRepository handles packaging integration operations
type IntegrationPackagingRepository interface {
	GetIntegrationUpdateByDate(date time.Time) ([]IntegrationPackaging, error)
	GetIntegrationsByCodIbm(codigoIbm string, transactionId string) ([]IntegrationPackaging, error)
	GetIntegrations(ip *IntegrationPackaging) ([]IntegrationPackaging, error)
	GetTransactionByRemove(date time.Time) ([]IntegrationPackaging, error)
	RemoveById(id int) error
	MoveIntegrationPackagingStaging(dataCorte time.Time) error
	ClearIntegrationPackagingByDealer(idDealer int) error
	ClearIntegrationPackagingByCutOffDate(cutOffDate time.Time, doPurge string) error
}

// IntegrationComboRepository handles combo integration operations
type IntegrationComboRepository interface {
	GetIntegrationUpdateComboByDate(date time.Time) ([]IntegrationCombo, error)
	GetIntegrationComboByDate(date time.Time) ([]IntegrationCombo, error)
	RemoveIntegrationCombo(date time.Time, doPurge string) error
	MoveIntegrationComboStaging(dataCorte time.Time) error
}
