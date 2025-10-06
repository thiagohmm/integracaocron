package usecases

import (
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/thiagohmm/integracaocron/domain/entities"
)

// IntegrationJobUseCase handles the main integration job operations
type IntegrationJobUseCase struct {
	parameterRepo   entities.ParameterRepository
	integrationRepo entities.IntegrationRepository
	networkRepo     entities.NetworkRepository
	db              *sql.DB
}

// NewIntegrationJobUseCase creates a new instance of IntegrationJobUseCase
func NewIntegrationJobUseCase(
	parameterRepo entities.ParameterRepository,
	integrationRepo entities.IntegrationRepository,
	networkRepo entities.NetworkRepository,
	db *sql.DB,
) *IntegrationJobUseCase {
	return &IntegrationJobUseCase{
		parameterRepo:   parameterRepo,
		integrationRepo: integrationRepo,
		networkRepo:     networkRepo,
		db:              db,
	}
}

// ProductNetworkMain is the Go equivalent of the main TypeScript function
func (uc *IntegrationJobUseCase) ProductNetworkMain(dataCorte time.Time) error {
	log.Println("Job Integração - Início")

	// Begin transaction
	tx, err := uc.db.Begin()
	if err != nil {
		return fmt.Errorf("erro ao iniciar transação: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p) // re-panic after rollback
		}
	}()

	// Execute all integration jobs
	if err := uc.IntegrationJob(); err != nil {
		tx.Rollback()
		return fmt.Errorf("erro no integration job: %w", err)
	}

	if err := uc.ReplicateNetworkProductsJob(); err != nil {
		tx.Rollback()
		return fmt.Errorf("erro no replicate network products job: %w", err)
	}

	if err := uc.MoveDataJob(dataCorte); err != nil {
		tx.Rollback()
		return fmt.Errorf("erro no move data job: %w", err)
	}

	if err := uc.UpdateExpirationSlaRequestsJob(); err != nil {
		tx.Rollback()
		return fmt.Errorf("erro no update expiration SLA requests job: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("erro ao fazer commit da transação: %w", err)
	}

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
	log.Println("Replicar produtos redes - Início.")

	networks, err := uc.networkRepo.GetNetwork()
	if err != nil {
		return fmt.Errorf("erro ao obter redes: %w", err)
	}

	for _, net := range networks {
		lojas, err := uc.networkRepo.ListByAllByIdDealerNew(net.IdRevendedor)
		if err != nil {
			log.Printf("Erro ao obter lojas para revendedor %d: %v", net.IdRevendedor, err)
			continue
		}

		err = uc.networkRepo.ReplicateProductNetwork(net.IdRede)
		if err != nil {
			log.Printf("Erro ao replicar produtos da rede %d: %v", net.IdRede, err)
			continue
		}

		for _, ljsItem := range lojas {
			_, err := uc.networkRepo.GetNetworkReplicadosByDealer(ljsItem.IdRevendedor)
			if err != nil {
				log.Printf("Erro ao obter replicados do revendedor %d: %v", ljsItem.IdRevendedor, err)
				continue
			}

			_, err = uc.networkRepo.GetProductsByReplicateNetworkServiceNew(ljsItem.IdRevendedor)
			if err != nil {
				log.Printf("Erro ao obter produtos para replicação do revendedor %d: %v", ljsItem.IdRevendedor, err)
				continue
			}
		}
	}

	log.Println("Replicar produtos redes - Fim.")
	return nil
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
