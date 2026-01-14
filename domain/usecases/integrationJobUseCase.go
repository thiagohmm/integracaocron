package usecases

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/thiagohmm/integracaocron/domain/entities"
	"github.com/thiagohmm/integracaocron/pkg/tracing"
)

// IntegrationJobUseCase handles the main integration job operations
type IntegrationJobUseCase struct {
	parameterRepo        entities.ParameterRepository
	integrationRepo      entities.IntegrationRepository
	networkRepo          entities.NetworkRepository
	productIntegrationUC *ProductIntegrationUseCase
	db                   *sql.DB
}

// NewIntegrationJobUseCase creates a new instance of IntegrationJobUseCase
func NewIntegrationJobUseCase(
	parameterRepo entities.ParameterRepository,
	integrationRepo entities.IntegrationRepository,
	networkRepo entities.NetworkRepository,
	productIntegrationUC *ProductIntegrationUseCase,
	db *sql.DB,
) *IntegrationJobUseCase {
	return &IntegrationJobUseCase{
		parameterRepo:        parameterRepo,
		integrationRepo:      integrationRepo,
		networkRepo:          networkRepo,
		productIntegrationUC: productIntegrationUC,
		db:                   db,
	}
}

// ProductNetworkMain is the Go equivalent of the main TypeScript function
func (uc *IntegrationJobUseCase) ProductNetworkMain(dataCorte time.Time) error {
	ctx := context.Background()
	ctx, span := tracing.StartSpan(ctx, "ProductNetworkMain")
	defer span.End()

	log.Println("Job Integração - Início")

	// 🔍 Tracing: Registrar data de corte
	tracing.AddStringAttribute(ctx, "data_corte", dataCorte.Format(time.RFC3339))

	// Begin transaction
	tx, err := uc.db.Begin()
	if err != nil {
		tracing.RecordError(ctx, err)
		return fmt.Errorf("erro ao iniciar transação: %w", err)
	}

	// ✅ CORREÇÃO CRÍTICA: Garantir que transação SEMPRE seja fechada
	committed := false
	defer func() {
		if !committed {
			if rbErr := tx.Rollback(); rbErr != nil {
				log.Printf("Erro ao fazer rollback da transação: %v", rbErr)
			}
		}
		if r := recover(); r != nil {
			log.Printf("PANIC recuperado em ProductNetworkMain: %v", r)
			panic(r) // re-panic after cleanup
		}
	}()

	// Execute all integration jobs
	if err := uc.IntegrationJob(); err != nil {
		log.Printf("Erro no integration job: %v", err)
		tracing.RecordError(ctx, err)
		tracing.AddEvent(ctx, "integration_job.failed", tracing.StringAttr("error", err.Error()))
		return fmt.Errorf("erro no integration job: %w", err)
	}
	tracing.AddEvent(ctx, "integration_job.completed")

	if err := uc.ReplicateNetworkProductsJob(); err != nil {
		log.Printf("Erro no replicate network products job: %v", err)
		tracing.RecordError(ctx, err)
		tracing.AddEvent(ctx, "replicate_network.failed", tracing.StringAttr("error", err.Error()))
		return fmt.Errorf("erro no replicate network products job: %w", err)
	}
	tracing.AddEvent(ctx, "replicate_network.completed")

	if err := uc.MoveDataJob(dataCorte); err != nil {
		log.Printf("Erro no move data job: %v", err)
		tracing.RecordError(ctx, err)
		tracing.AddEvent(ctx, "move_data.failed", tracing.StringAttr("error", err.Error()))
		return fmt.Errorf("erro no move data job: %w", err)
	}
	tracing.AddEvent(ctx, "move_data.completed")

	if err := uc.UpdateExpirationSlaRequestsJob(); err != nil {
		log.Printf("Erro no update expiration SLA requests job: %v", err)
		tracing.RecordError(ctx, err)
		tracing.AddEvent(ctx, "update_expiration.failed", tracing.StringAttr("error", err.Error()))
		return fmt.Errorf("erro no update expiration SLA requests job: %w", err)
	}
	tracing.AddEvent(ctx, "update_expiration.completed")

	// Commit transaction
	if err := tx.Commit(); err != nil {
		tracing.RecordError(ctx, err)
		return fmt.Errorf("erro ao fazer commit da transação: %w", err)
	}
	committed = true

	// 🔍 Tracing: Registrar sucesso
	tracing.SetStatus(ctx, 1, "All integration jobs completed successfully") // codes.Ok = 1

	log.Println("Job Integração - Término")
	return nil
}

// FormatDateForOracle formats a Go time.Time to Oracle timestamp format
func (uc *IntegrationJobUseCase) FormatDateForOracle(date time.Time) string {
	// Oracle format: 'YYYY-MM-DD HH24:MI:SS.FF TZH:TZM'
	return date.Format("2006-01-02 15:04:05.000 -07:00")
}

// IntegrationJob handles the main integration cleanup and expiry operations
func (uc *IntegrationJobUseCase) IntegrationJob() error {
	log.Println("Remover Transação - Início")

	dataCorte := time.Now()
	dataCorteExpurgo := time.Now()

	// Get parameter for transaction removal
	paramJob, err := uc.GetValueParameterRemoveTransactionJob()
	if err != nil {
		return fmt.Errorf("erro ao obter parâmetro de remoção de transação: %w", err)
	}

	log.Printf("Param Job: %+v", paramJob)

	if paramJob == nil {
		log.Printf("Remover Transação - Não executada, função desligada, parâmetro nil")
		return nil
	}

	min, err := strconv.Atoi(paramJob.Valor)
	if err != nil {
		return fmt.Errorf("erro ao converter parâmetro para int: %w", err)
	}
	log.Printf("Min: %d", min)

	// Subtract minutes from current time
	dataCorte = dataCorte.Add(-time.Duration(min) * time.Minute)

	// Remove transactions
	if err := uc.RemoverTransacaoIntegracaoCombo(dataCorte); err != nil {
		return err
	}
	if err := uc.RemoverTransacaoIntegracaoEmbalagem(dataCorte); err != nil {
		return err
	}
	if err := uc.RemoverTransacaoIntegracaoEstruturaMercadologica(dataCorte); err != nil {
		return err
	}
	if err := uc.RemoverTransacaoIntegracaoProduto(dataCorte); err != nil {
		return err
	}
	if err := uc.RemoverTransacaoIntegracaoPromocao(dataCorte); err != nil {
		return err
	}

	// Update parameter
	if err := uc.SetValueParameterEndTransactionJob(); err != nil {
		return err
	}

	// Get expiry parameter
	paramExpurgo, err := uc.GetValueParameterExpurgoDiasJob()
	if err != nil {
		return fmt.Errorf("erro ao obter parâmetro de expurgo: %w", err)
	}

	log.Printf("Param Expurgo: %+v", paramExpurgo)

	dayExpurgo, err := strconv.Atoi(paramExpurgo.Valor)
	if err != nil {
		return fmt.Errorf("erro ao converter parâmetro de expurgo para int: %w", err)
	}
	log.Printf("Min Expurgo: %d", dayExpurgo)

	// Subtract days from current time
	dataCorteExpurgo = dataCorteExpurgo.AddDate(0, 0, -dayExpurgo)
	log.Printf("Data Corte Expurgo: %v", dataCorteExpurgo)

	// Execute expiry operations
	if err := uc.ExpurgoIntegracaoCombo(dataCorteExpurgo); err != nil {
		return err
	}
	if err := uc.ExpurgoIntegracaoEmbalagem(dataCorteExpurgo); err != nil {
		return err
	}
	if err := uc.ExpurgoIntegracaoEstruturaMercadologica(dataCorteExpurgo); err != nil {
		return err
	}
	if err := uc.ExpurgoIntegracaoProduto(dataCorteExpurgo); err != nil {
		return err
	}
	if err := uc.ExpurgoIntegracaoPromocao(dataCorteExpurgo); err != nil {
		return err
	}

	if err := uc.SetValueParameterExpurgoUltimaExcucaoJob(); err != nil {
		return err
	}

	log.Println("Remover transação - Fim")
	return nil
}

// Expiry operations
func (uc *IntegrationJobUseCase) ExpurgoIntegracaoCombo(dataCorte time.Time) error {
	data, err := uc.integrationRepo.GetIntegrationUpdateComboByDate(dataCorte)
	if err != nil {
		return err
	}

	for _, item := range data {
		if err := uc.integrationRepo.DeleteIntegrationCombo(item.IdIntegracaoCombo); err != nil {
			log.Printf("Erro ao deletar combo %d: %v", item.IdIntegracaoCombo, err)
			// Continue with other items
		}
	}
	return nil
}

func (uc *IntegrationJobUseCase) ExpurgoIntegracaoEmbalagem(dataCorte time.Time) error {
	return uc.integrationRepo.ClearIntegrationPackagingByCutOffDate(dataCorte, "SIM")
}

func (uc *IntegrationJobUseCase) ExpurgoIntegracaoEstruturaMercadologica(dataCorte time.Time) error {
	return uc.integrationRepo.RemoverTransacaoIntegracaoEstruturaMercadologica(dataCorte, "SIM")
}

func (uc *IntegrationJobUseCase) ExpurgoIntegracaoProduto(dataCorte time.Time) error {
	return uc.integrationRepo.RemoverTransacaoIntegracaoProduto(dataCorte, "SIM")
}

func (uc *IntegrationJobUseCase) ExpurgoIntegracaoPromocao(dataCorte time.Time) error {
	return uc.integrationRepo.RemoverTransacaoIntegracaoPromocao(dataCorte, "SIM")
}

// Transaction removal operations
func (uc *IntegrationJobUseCase) RemoverTransacaoIntegracaoCombo(dataCorte time.Time) error {
	log.Println("RemoverTransacaoIntegracaoCombo - Início")
	err := uc.integrationRepo.RemoveIntegrationCombo(dataCorte, "SIM")
	if err != nil {
		return err
	}
	log.Println("RemoverTransacaoIntegracaoCombo - Término")
	return nil
}

func (uc *IntegrationJobUseCase) RemoverTransacaoIntegracaoEmbalagem(dataCorte time.Time) error {
	log.Println("Remover transação integração embalagem - Início")
	err := uc.integrationRepo.ClearIntegrationPackagingByCutOffDate(dataCorte)
	if err != nil {
		return err
	}
	log.Println("Remover transação integração embalagem - Término")
	return nil
}

func (uc *IntegrationJobUseCase) RemoverTransacaoIntegracaoEstruturaMercadologica(dataCorte time.Time) error {
	log.Println("Remover Transação Integração Estrutura Mercadológica - Início")
	err := uc.integrationRepo.RemoverTransacaoIntegracaoEstruturaMercadologica(dataCorte)
	if err != nil {
		return err
	}
	log.Println("Remover Transação Integração Estrutura Mercadológica - Fim")
	return nil
}

func (uc *IntegrationJobUseCase) RemoverTransacaoIntegracaoProduto(dataCorte time.Time) error {
	log.Println("Remover transação integração produto - Início")
	err := uc.integrationRepo.RemoverTransacaoIntegracaoProduto(dataCorte)
	if err != nil {
		return err
	}
	log.Println("Remover transação integração produto - Término")
	return nil
}

func (uc *IntegrationJobUseCase) RemoverTransacaoIntegracaoPromocao(dataCorte time.Time) error {
	log.Println("Remover transação integração promoção - Início")
	err := uc.integrationRepo.RemoverTransacaoIntegracaoPromocao(dataCorte)
	if err != nil {
		return err
	}
	log.Println("Remover transação integração promoção - Término")
	return nil
}

// Parameter operations
func (uc *IntegrationJobUseCase) GetValueParameterRemoveTransactionJob() (*entities.IParameter, error) {
	return uc.parameterRepo.ListByCodeParameter("REMOVER_TRANSACAO_MINUTOS")
}

func (uc *IntegrationJobUseCase) GetValueParameterExpurgoDiasJob() (*entities.IParameter, error) {
	return uc.parameterRepo.ListByCodeParameter("EXPURGO_INTEGRACAO_DIAS")
}

func (uc *IntegrationJobUseCase) SetValueParameterExpurgoUltimaExcucaoJob() error {
	param, err := uc.parameterRepo.ListByCodeParameter("Parametro_ExpurgoIntegracaoUltimaExecucao")
	if err != nil {
		return err
	}
	if param != nil && param.Ambiente == "*" {
		param.Valor = time.Now().String()
		return uc.parameterRepo.Update(param)
	}
	return nil
}

func (uc *IntegrationJobUseCase) SetValueParameterEndTransactionJob() error {
	param, err := uc.parameterRepo.ListByCodeParameter("RemoverTransacaoUltimaExecucao")
	if err != nil {
		return err
	}
	if param != nil && param.Ambiente == "*" {
		param.Valor = time.Now().String()
		return uc.parameterRepo.Update(param)
	}
	return nil
}

// ReplicateNetworkProductsJob replicates products across networks
func (uc *IntegrationJobUseCase) ReplicateNetworkProductsJob() error {
	ctx := context.Background()
	ctx, span := tracing.StartSpan(ctx, "ReplicateNetworkProductsJob")
	defer span.End()

	log.Println("Replicar produtos redes - Início.")

	networks, err := uc.networkRepo.GetNetwork()
	if err != nil {
		tracing.RecordError(ctx, err)
		return fmt.Errorf("erro ao obter redes: %w", err)
	}

	log.Printf("Processando %d redes para replicação", len(networks))
	tracing.AddIntAttribute(ctx, "total_networks", len(networks))

	for i, net := range networks {
		// 🔍 Tracing: Criar span para cada rede
		networkCtx, networkSpan := tracing.StartSpan(ctx, "ProcessNetwork")
		tracing.AddInt64Attribute(networkCtx, "network_id", int64(net.IdRede))
		tracing.AddInt64Attribute(networkCtx, "dealer_id", int64(net.IdRevendedor))
		tracing.AddIntAttribute(networkCtx, "network_index", i+1)

		log.Printf("[%d/%d] Processando rede %d (Revendedor: %d)", i+1, len(networks), net.IdRede, net.IdRevendedor)

		// Replicar produtos da rede ANTES de buscar lojas (otimização)
		err = uc.networkRepo.ReplicateProductNetwork(net.IdRede)
		if err != nil {
			log.Printf("Erro ao replicar produtos da rede %d: %v", net.IdRede, err)
			tracing.RecordError(networkCtx, err)
			tracing.AddEvent(networkCtx, "replicate_products.failed", tracing.StringAttr("error", err.Error()))
			networkSpan.End()
			continue
		}
		tracing.AddEvent(networkCtx, "replicate_products.completed")

		// Buscar lojas da rede
		lojas, err := uc.networkRepo.ListByAllByIdDealerNew(net.IdRevendedor)
		if err != nil {
			log.Printf("Erro ao obter lojas para revendedor %d: %v", net.IdRevendedor, err)
			tracing.RecordError(networkCtx, err)
			tracing.AddEvent(networkCtx, "list_dealers.failed", tracing.StringAttr("error", err.Error()))
			networkSpan.End()
			continue
		}

		log.Printf("Rede %d: encontradas %d lojas para processar", net.IdRede, len(lojas))
		tracing.AddIntAttribute(networkCtx, "dealers_count", len(lojas))

		// Processar lojas em batch (otimização)
		if len(lojas) > 0 {
			// Extrair IDs das lojas para processar em batch
			var dealerIDs []int
			for _, loja := range lojas {
				dealerIDs = append(dealerIDs, loja.IdRevendedor)
			}

			// Processar produtos em batch (reduz queries)
			err := uc.networkRepo.ProcessReplicatedProductsInBatch(dealerIDs, net.IdRede)
			if err != nil {
				log.Printf("Erro ao processar produtos em batch para rede %d: %v", net.IdRede, err)
				tracing.RecordError(networkCtx, err)
				tracing.AddEvent(networkCtx, "batch_processing.failed", tracing.StringAttr("error", err.Error()))
				// Fallback: processar individualmente
				uc.processLojasIndividually(lojas, net.IdRede)
				tracing.AddEvent(networkCtx, "fallback_processing.completed")
			} else {
				tracing.AddEvent(networkCtx, "batch_processing.completed", tracing.IntAttr("dealers_processed", len(dealerIDs)))
			}
		}

		log.Printf("Rede %d processada com sucesso", net.IdRede)
		tracing.SetStatus(networkCtx, 1, "Network processed successfully") // codes.Ok = 1
		networkSpan.End()
	}

	// 🔍 Tracing: Finalizar job com sucesso
	tracing.SetStatus(ctx, 1, "All networks replicated successfully") // codes.Ok = 1
	log.Println("Replicar produtos redes - Fim.")
	return nil
}

// processLojasIndividually fallback method to process stores individually
func (uc *IntegrationJobUseCase) processLojasIndividually(lojas []entities.DealerNetwork, idRede int) {
	log.Printf("Processando %d lojas individualmente (fallback) para rede %d", len(lojas), idRede)

	for _, loja := range lojas {
		products, err := uc.networkRepo.GetProductsByReplicateNetworkServiceNew(loja.IdRevendedor)
		if err != nil {
			log.Printf("Erro ao obter produtos para revendedor %d: %v", loja.IdRevendedor, err)
			continue
		}

		for _, product := range products {
			productSelect := entities.ProductSelectIntegration{
				CodRMS: entities.FlexibleString(fmt.Sprintf("%d", product.CodigoRMS)),
			}
			err := uc.productIntegrationUC.IntegrateProductService(product.IdProduto, loja.IdRevendedor, productSelect)
			if err != nil {
				log.Printf("Erro ao integrar produto %d para revendedor %d: %v", product.IdProduto, loja.IdRevendedor, err)
			}
		}
	}
}

// MoveDataJob moves data between staging tables
func (uc *IntegrationJobUseCase) MoveDataJob(dataCorte time.Time) error {
	if err := uc.MoverEstruturaMercadologica(dataCorte); err != nil {
		return err
	}
	if err := uc.MoverProduto(dataCorte); err != nil {
		return err
	}
	if err := uc.MoverEmbalagem(dataCorte); err != nil {
		return err
	}
	if err := uc.MoverCombo(dataCorte); err != nil {
		return err
	}
	if err := uc.MoverPromocao(dataCorte); err != nil {
		return err
	}
	return nil
}

func (uc *IntegrationJobUseCase) MoverEstruturaMercadologica(dataCorte time.Time) error {
	return uc.integrationRepo.MoveIntegrationMarketingStructure(dataCorte)
}

func (uc *IntegrationJobUseCase) MoverProduto(dataCorte time.Time) error {
	return uc.integrationRepo.MoveIntegrationProductStaging(dataCorte)
}

func (uc *IntegrationJobUseCase) MoverEmbalagem(dataCorte time.Time) error {
	return uc.integrationRepo.MoveIntegrationPackagingStaging(dataCorte)
}

func (uc *IntegrationJobUseCase) MoverCombo(dataCorte time.Time) error {
	return uc.integrationRepo.MoveIntegrationComboStaging(dataCorte)
}

func (uc *IntegrationJobUseCase) MoverPromocao(dataCorte time.Time) error {
	return uc.integrationRepo.MoveIntegrationPromotionStaging(dataCorte)
}

// UpdateExpirationSlaRequestsJob updates expired SLA requests
func (uc *IntegrationJobUseCase) UpdateExpirationSlaRequestsJob() error {
	return uc.integrationRepo.UpdateExpiredSlaSolicitation()
}
