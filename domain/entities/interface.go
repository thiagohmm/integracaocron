package entities

import "time"

// ProductIntegrationRepository handles product integration database operations
type ProductIntegrationRepository interface {
	GetIntegrRmsProductsIn() ([]IntegrRmsProductIn, error)
	RemoveProductService(rms IntegrRmsProductIn) error
	GetMarketingStructureLevel2(idLevel2 int) (*MarketingStructure, error)
	GetMarketingStructureLevel4(idLevel4 int) ([]MarketingStructure, error)
	GetBrandByIndustryName(brandName, industryName string) ([]Brand, error)
	GetIndustryByNameAndStatus(nomeIndustria string, statusIndustria int) (*Industry, error)
	SaveIndustry(industry Industry) (*Industry, error)
	SaveBrand(brand Brand) (*Brand, error)
	GetProductByCodeRMS(codeRms int) (*Product, error)
	GetProductPackagingByBarCode(barCode string) (*ProductPackaging, error)
	GetDepartmentNameByID(id *int) (string, error)
	GetSectionNameByID(id *int) (*Section, error)
	GetBrandDescByID(id *int) ([]Brand, error)
	DoPackageProductIntegration(iprID int) (*LogValidate, error)
	SaveLogIntegration(log LogIntegrRMS) error
	SendToQueue(message QueueMessage) error
	RemoverCaracteresEspeciais(input string) string
	ValidateMarketingStructureLevel2(ms *MarketingStructure) *LogValidate
	ValidateBrandDesc(descMarca string) *LogValidate
	AddOrUpdateStaging(idProduto, idDealer int, jsonPayload string) error
	ValidateIndustry(industry string) *LogValidate
	GetAllProductIntegrationStagingRecords() ([]ProductIntegrationStaging, error)
	InsertProduct(product *ProductNew) (int, error)
	UpdateProduct(product *ProductNew) error
	InsertProductPackaging(productID int, pkg ProductPackaging) error
	DeleteProductPackagingByProductID(productID int) error
	RemoveProductIntegrationStagingRecord(id int) error
}

// PromotionRepository handles system parameters
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

// IntegrationComboRepository handles combo integration retrieval and movement
type IntegrationComboRepository interface {
	GetIntegrationUpdateComboByDate(date time.Time) ([]IntegrationCombo, error)
	GetIntegrationComboByDate(date time.Time) ([]IntegrationCombo, error)
	RemoveIntegrationCombo(date time.Time, doPurge string) error
	MoveIntegrationComboStaging(dataCorte time.Time) error
}

// IntegrationMarketingStructureRepository handles marketing structure integrations
type IntegrationMarketingStructureRepository interface {
	GetIntegrationUpdateByDate(date time.Time) ([]IntegrationMarketingStructure, error)
	GetIntegrations(ip *IntegrationMarketingStructure) ([]IntegrationMarketingStructure, error)
	GetIMSByDate(date time.Time) ([]IntegrationMarketingStructure, error)
	RemoveById(id int) error
	RemoverTransacaoIntegracaoEstruturaMercadologica(dataCorte time.Time, fazExpurgo string) error
}

// IntegrationPackagingRepository handles packaging integrations
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

// IntegrationRepository handles general integration operations
type IntegrationRepository interface {
	RemoveIntegrationCombo(dataCorte time.Time, expurgo ...string) error
	ClearIntegrationPackagingByCutOffDate(dataCorte time.Time, expurgo ...string) error
	RemoverTransacaoIntegracaoEstruturaMercadologica(dataCorte time.Time, expurgo ...string) error
	RemoverTransacaoIntegracaoProduto(dataCorte time.Time, expurgo ...string) error
	RemoverTransacaoIntegracaoPromocao(dataCorte time.Time, expurgo ...string) error
	CheckMarketingStructure() (bool, error)
	CheckProductIntegration() (bool, error)
	CheckPackagingIntegration() (bool, error)
	CheckComboIntegration() (bool, error)
	CheckPromotionIntegration() (bool, error)
	HasMarketingStructureStaging() (bool, error)
	HasProductStaging() (bool, error)
	HasPackagingStaging() (bool, error)
	HasComboStaging() (bool, error)
	HasPromotionStaging() (bool, error)
	MoveIntegrationMarketingStructure(dataCorte time.Time) error
	MoveIntegrationProductStaging(dataCorte time.Time) error
	MoveIntegrationPackagingStaging(dataCorte time.Time) error
	MoveIntegrationComboStaging(dataCorte time.Time) error
	MoveIntegrationPromotionStaging(dataCorte time.Time) error
	GetIntegrationUpdateComboByDate(dataCorte time.Time) ([]IntegrationCombo, error)
	DeleteIntegrationCombo(idIntegracaoCombo int) error
	UpdateExpiredSlaSolicitation() error
}

// NetworkRepository handles network-related operations
type NetworkRepository interface {
	GetNetwork() ([]Network, error)
	ListByAllByIdDealerNew(idDealer int) ([]DealerNetwork, error)
	ReplicateProductNetwork(idRede int) error
	ProcessReplicatedProductsInBatch(dealerIDs []int, idRede int) error
	GetNetworkReplicadosByDealer(idRevendedor int) ([]interface{}, error)
	GetProductsByReplicateNetworkServiceNew(idRevendedor int) ([]ProductSelect, error)
	GetProductsByReplicateNetworkReplicate(idProduto int) ([]ProductSelect, error)
	GetProductsByReplicateNetworkNew(idRevendedor int, idProduto *int, produtosReplicados string) error
}
