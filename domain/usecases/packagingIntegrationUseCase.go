package usecases

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/thiagohmm/integracaocron/domain/entities"
	"github.com/thiagohmm/integracaocron/domain/repositories"
)

// PackagingIntegrationUseCase handles packaging integration business logic
type PackagingIntegrationUseCase struct {
	productRepo         *repositories.ProductRepository
	productPackagingRepo *repositories.ProductPackagingRepository
	unitOfMeasurementRepo *repositories.UnitOfMeasurementRepository
	stagingRepo         *repositories.PackagingIntegrationStagingRepository
}

// NewPackagingIntegrationUseCase creates a new instance of PackagingIntegrationUseCase
func NewPackagingIntegrationUseCase(
	productRepo *repositories.ProductRepository,
	productPackagingRepo *repositories.ProductPackagingRepository,
	unitOfMeasurementRepo *repositories.UnitOfMeasurementRepository,
	stagingRepo *repositories.PackagingIntegrationStagingRepository,
) *PackagingIntegrationUseCase {
	return &PackagingIntegrationUseCase{
		productRepo:         productRepo,
		productPackagingRepo: productPackagingRepo,
		unitOfMeasurementRepo: unitOfMeasurementRepo,
		stagingRepo:         stagingRepo,
	}
}

// PackagingIntegrateService integrates packaging for a given product and dealer
func (uc *PackagingIntegrationUseCase) PackagingIntegrateService(idProduct, idDealer int) error {
	embs, err := uc.createPackaging(idProduct)
	if err != nil {
		return fmt.Errorf("error creating packaging: %w", err)
	}

	for _, emb := range embs {
		jsonPayload, err := json.Marshal(emb.Mix)
		if err != nil {
			log.Printf("error marshalling packaging mix: %v", err)
			continue
		}

		err = uc.stagingRepo.AddOrUpdate(emb.IdEmbalagemProduto, idDealer, string(jsonPayload))
		if err != nil {
			log.Printf("error adding or updating packaging staging: %v", err)
			continue
		}
	}

	return nil
}

func (uc *PackagingIntegrationUseCase) createPackaging(idProduto int) ([]entities.RetPackaging, error) {
	var ret []entities.RetPackaging

	prod, err := uc.productRepo.GetProductByID(idProduto)
	if err != nil {
		return nil, fmt.Errorf("error getting product by ID: %w", err)
	}
	if prod == nil {
		return nil, fmt.Errorf("product not found for ID: %d", idProduto)
	}

	productPackaging, err := uc.productPackagingRepo.ListByIdProductPackaging(idProduto)
	if err != nil {
		return nil, fmt.Errorf("error getting product packaging: %w", err)
	}

	var packagingMain *entities.ProductPackaging
	for i, p := range productPackaging {
		if p.QuantidadeEmbalagem == 1 {
			packagingMain = &productPackaging[i]
			break
		}
	}

	for _, item := range productPackaging {
		var unid *entities.UnitOfMeasurement
		if item.IdUnidadeMedida != nil {
			unid, err = uc.unitOfMeasurementRepo.GetUnitOfMeasurementByID(*item.IdUnidadeMedida)
			if err != nil {
				log.Printf("error getting unit of measurement: %v", err)
			}
		}

		var codUnid string
		if unid != nil {
			codUnid = unid.CodigoUnidadeMedida
		}

		var codBarraMain string
		if packagingMain != nil {
			codBarraMain = packagingMain.CodigoBarras
		}

		dto := entities.Mix{
			CodMix:  item.CodigoBarras,
			DescMix: fmt.Sprintf("%s - %d %s - %s", prod.DescricaoProduto, item.QuantidadeEmbalagem, codUnid, codBarraMain),
			DataDe:  time.Now(),
			DataAte: time.Now(),
			Grupos:  []entities.GroupItemMix{},
			DtAlt:   time.Now(),
		}
		if prod.Ativo == 1 {
			dto.Status = "A"
		} else {
			dto.Status = "I"
		}

		grup := entities.GroupItemMix{
			Desc:     "EMBALAGEM",
			QtdeItem: item.QuantidadeEmbalagem,
			Items:    []entities.ItemGroupMix{},
		}

		grup.Items = append(grup.Items, entities.ItemGroupMix{
			CodItem:  fmt.Sprintf("%d", *prod.IdProduto),
			CodBarra: codBarraMain,
		})

		dto.Grupos = append(dto.Grupos, grup)

		ret = append(ret, entities.RetPackaging{
			IdEmbalagemProduto: *item.IdProduto,
			Mix:                dto,
		})
	}

	return ret, nil
}
