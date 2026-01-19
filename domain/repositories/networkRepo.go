package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/thiagohmm/integracaocron/domain/entities"
)

// NetworkRepositoryImpl implements the NetworkRepository interface
type NetworkRepositoryImpl struct {
	db *sql.DB
}

// NewNetworkRepository creates a new instance of NetworkRepository
func NewNetworkRepository(db *sql.DB) entities.NetworkRepository {
	return &NetworkRepositoryImpl{
		db: db,
	}
}

// GetNetwork retrieves all networks with replication enabled
func (r *NetworkRepositoryImpl) GetNetwork() ([]entities.Network, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT IDREDE, DESCRICAOREDE, IDREVENDEDOR, STATUSREDE, REPLICARPRODUTO, 
			   DATACADASTRO, DATAATUALIZACAO, PERMITEREPLICARPRODUTO, USUARIOREPLICOU
		FROM REDE 
		WHERE PERMITEREPLICARPRODUTO = '1' 
		  AND STATUSREDE = '1' 
		  AND REPLICARPRODUTO = '1'
		ORDER BY IDREDE`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Erro ao consultar redes: %v", err)
		return nil, fmt.Errorf("erro ao consultar redes: %w", err)
	}
	defer rows.Close()

	var networks []entities.Network
	for rows.Next() {
		var network entities.Network
		err := rows.Scan(
			&network.IdRede,
			&network.DescricaoRede,
			&network.IdRevendedor,
			&network.StatusRede,
			&network.ReplicarProduto,
			&network.DataCadastro,
			&network.DataAtualizacao,
			&network.PermiteReplicarProduto,
			&network.UsuarioReplicou,
		)
		if err != nil {
			log.Printf("Erro ao escanear rede: %v", err)
			continue
		}
		networks = append(networks, network)
	}

	log.Printf("Encontradas %d redes com replicação habilitada", len(networks))
	return networks, nil
}

// ListByAllByIdDealerNew retrieves dealers by ID based on network principal dealer
func (r *NetworkRepositoryImpl) ListByAllByIdDealerNew(idDealer int) ([]entities.DealerNetwork, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `
		SELECT revendedorrede.idrevendedor "IdRevendedor"
		FROM RevendedorRede revendedorrede
		JOIN revendedor revendedor ON (revendedor.idrevendedor = revendedorrede.idrevendedor)
		JOIN rede rede ON (rede.idrede = revendedorrede.idrede)
		WHERE 1 = 1
		AND rede.idrevendedor = :1
		ORDER BY revendedorrede.idrevendedor`

	rows, err := r.db.QueryContext(ctx, query, idDealer)
	if err != nil {
		log.Printf("Erro ao consultar revendedores da rede por dealer %d: %v", idDealer, err)
		return nil, fmt.Errorf("erro ao consultar revendedores da rede: %w", err)
	}
	defer rows.Close()

	var dealers []entities.DealerNetwork
	for rows.Next() {
		var dealer entities.DealerNetwork
		err := rows.Scan(&dealer.IdRevendedor)
		if err != nil {
			log.Printf("Erro ao escanear revendedor: %v", err)
			continue
		}
		dealers = append(dealers, dealer)
	}

	log.Printf("Encontrados %d revendedores na rede do dealer principal %d", len(dealers), idDealer)
	return dealers, nil
}

// ReplicateProductNetwork replicates products for a network using stored procedure
func (r *NetworkRepositoryImpl) ReplicateProductNetwork(idRede int) error {
	// ✅ CORREÇÃO: Usar stored procedure ao invés de placeholder
	return r.ReplicateProductNetworkSP(idRede)
}

// GetNetworkReplicadosByDealer gets replicated data by dealer (limited to first row)
func (r *NetworkRepositoryImpl) GetNetworkReplicadosByDealer(idRevendedor int) ([]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT IdProduto 
		FROM ProdutosReplicados 
		WHERE IdRevendedor = :1 
		FETCH FIRST 1 ROW ONLY`

	rows, err := r.db.QueryContext(ctx, query, idRevendedor)
	if err != nil {
		log.Printf("Erro ao consultar produtos replicados por dealer: %v", err)
		return nil, fmt.Errorf("erro ao consultar produtos replicados: %w", err)
	}
	defer rows.Close()

	var results []interface{}
	for rows.Next() {
		var idProduto int
		err := rows.Scan(&idProduto)
		if err != nil {
			log.Printf("Erro ao escanear produto replicado: %v", err)
			continue
		}
		// Create a map to match the expected interface{} return type
		result := map[string]interface{}{
			"IdProduto": idProduto,
		}
		results = append(results, result)
	}

	log.Printf("Encontrados %d produtos replicados para dealer %d", len(results), idRevendedor)
	return results, nil
}

// GetProductsByReplicateNetworkServiceNew gets products for replication
func (r *NetworkRepositoryImpl) GetProductsByReplicateNetworkServiceNew(idRevendedor int) ([]entities.ProductSelect, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `SELECT Cod, CODIGO_RMS, ID_PRODUTO FROM Produtos WHERE IdRevendedor = :1 AND StatusReplicacao = 'PENDENTE'`

	rows, err := r.db.QueryContext(ctx, query, idRevendedor)
	if err != nil {
		log.Printf("Erro ao consultar produtos para replicação: %v", err)
		return nil, fmt.Errorf("erro ao consultar produtos para replicação: %w", err)
	}
	defer rows.Close()

	var products []entities.ProductSelect
	for rows.Next() {
		var product entities.ProductSelect
		err := rows.Scan(&product.Cod, &product.CodigoRMS, &product.IdProduto)
		if err != nil {
			log.Printf("Erro ao escanear produto para replicação: %v", err)
			continue
		}
		products = append(products, product)
	}

	log.Printf("Encontrados %d produtos para replicação do revendedor %d", len(products), idRevendedor)
	return products, nil
}

// GetProductsByReplicateNetworkReplicate gets products by replication network (legacy method)
func (r *NetworkRepositoryImpl) GetProductsByReplicateNetworkReplicate(idProduto int) ([]entities.ProductSelect, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `SELECT Cod, CODIGO_RMS, ID_PRODUTO FROM Produtos WHERE IdProduto = :1 AND StatusReplicacao = 'ATIVO'`

	rows, err := r.db.QueryContext(ctx, query, idProduto)
	if err != nil {
		log.Printf("Erro ao consultar produtos por replicação de rede: %v", err)
		return nil, fmt.Errorf("erro ao consultar produtos por replicação: %w", err)
	}
	defer rows.Close()

	var products []entities.ProductSelect
	for rows.Next() {
		var product entities.ProductSelect
		err := rows.Scan(&product.Cod, &product.CodigoRMS, &product.IdProduto)
		if err != nil {
			log.Printf("Erro ao escanear produto para replicação de rede: %v", err)
			continue
		}
		products = append(products, product)
	}

	log.Printf("Encontrados %d produtos para replicação de rede do produto %d", len(products), idProduto)
	return products, nil
}

// GetProductsByReplicateNetworkNew calls stored procedure to save product integration
// Equivalent to TypeScript: getProductsByReplicateNetworkNew
func (r *NetworkRepositoryImpl) GetProductsByReplicateNetworkNew(idRevendedor int, idProduto *int, produtosReplicados string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	query := `BEGIN SP_GRAVARINTEGRACAOPRODUTOSTAGINGPARCIAL(:1, :2, :3); END;`

	_, err := r.db.ExecContext(ctx, query, idRevendedor, idProduto, produtosReplicados)
	if err != nil {
		log.Printf("Erro ao executar SP_GRAVARINTEGRACAOPRODUTOSTAGINGPARCIAL: %v", err)
		return fmt.Errorf("erro ao gravar integração produto staging parcial: %w", err)
	}

	return nil
}

// ProcessReplicatedProductsInBatch processes products in batch for better performance
func (r *NetworkRepositoryImpl) ProcessReplicatedProductsInBatch(dealerIDs []int, idRede int) error {
	if len(dealerIDs) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second) // Timeout maior para batch
	defer cancel()

	// Usar stored procedure ou query otimizada para processar em batch
	// Isso evita N queries individuais
	log.Printf("Processando batch de %d revendedores para rede %d", len(dealerIDs), idRede)

	// Opção 1: Usar stored procedure (mais eficiente)
	query := `BEGIN sp_ProcessarProdutosRede(:1, :2); END;`

	// Converter array de IDs para string separada por vírgula
	dealerIDsStr := ""
	for i, id := range dealerIDs {
		if i > 0 {
			dealerIDsStr += ","
		}
		dealerIDsStr += fmt.Sprintf("%d", id)
	}

	_, err := r.db.ExecContext(ctx, query, dealerIDsStr, idRede)
	if err != nil {
		// Se a stored procedure não existir, fazer fallback
		if strings.Contains(err.Error(), "ORA-00904") || strings.Contains(err.Error(), "PLS-00201") {
			log.Printf("Stored procedure sp_ProcessarProdutosRede não encontrada, usando fallback")
			return r.processReplicatedProductsInBatchFallback(dealerIDs, idRede)
		}
		log.Printf("Erro ao processar produtos em batch: %v", err)
		return fmt.Errorf("erro ao processar produtos em batch: %w", err)
	}

	log.Printf("Batch processado com sucesso: %d revendedores", len(dealerIDs))
	return nil
}

// processReplicatedProductsInBatchFallback fallback method using SQL directly
func (r *NetworkRepositoryImpl) processReplicatedProductsInBatchFallback(dealerIDs []int, idRede int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Construir query com IN clause (mais eficiente que N queries)
	dealerIDsStr := ""
	for i, id := range dealerIDs {
		if i > 0 {
			dealerIDsStr += ","
		}
		dealerIDsStr += fmt.Sprintf("%d", id)
	}

	// Query otimizada que processa todos os revendedores de uma vez
	// Nota: INTEGRACAOPRODUTOSTAGING não possui coluna IDREDE, apenas IDREVENDEDOR
	// Os revendedores já estão filtrados pela rede antes de chegar aqui
	query := fmt.Sprintf(`
		UPDATE INTEGRACAOPRODUTOSTAGING 
		SET DATAPROCESSAMENTO = SYSTIMESTAMP,
			STATUSPROCESSAMENTO = 1
		WHERE IDREVENDEDOR IN (%s)`, dealerIDsStr)

	result, err := r.db.ExecContext(ctx, query)
	if err != nil {
		log.Printf("Erro ao processar produtos (fallback): %v", err)
		return fmt.Errorf("erro ao processar produtos (fallback): %w", err)
	}

	rows, _ := result.RowsAffected()
	log.Printf("Batch fallback processado: %d produtos atualizados", rows)
	return nil
}

// GetNetworkByDealer retrieves a network by dealer ID
func (r *NetworkRepositoryImpl) GetNetworkByDealer(idDealer int) (*entities.Network, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		SELECT IDREDE, DESCRICAOREDE, IDREVENDEDOR, STATUSREDE, REPLICARPRODUTO, 
			   DATACADASTRO, DATAATUALIZACAO, PERMITEREPLICARPRODUTO, USUARIOREPLICOU
		FROM REDE 
		WHERE IDREVENDEDOR = :1`

	var network entities.Network
	err := r.db.QueryRowContext(ctx, query, idDealer).Scan(
		&network.IdRede,
		&network.DescricaoRede,
		&network.IdRevendedor,
		&network.StatusRede,
		&network.ReplicarProduto,
		&network.DataCadastro,
		&network.DataAtualizacao,
		&network.PermiteReplicarProduto,
		&network.UsuarioReplicou,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			log.Printf("Rede não encontrada para revendedor: %d", idDealer)
			return nil, nil // Return nil instead of error for not found
		}
		log.Printf("Erro ao consultar rede por revendedor %d: %v", idDealer, err)
		return nil, fmt.Errorf("erro ao consultar rede: %w", err)
	}

	log.Printf("Rede encontrada para revendedor %d: %+v", idDealer, network)
	return &network, nil
}

// UpdateNetwork updates a network
func (r *NetworkRepositoryImpl) UpdateNetwork(network *entities.Network) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		UPDATE REDE SET 
			DESCRICAOREDE = :1, 
			IDREVENDEDOR = :2, 
			STATUSREDE = :3, 
			REPLICARPRODUTO = :4, 
			DATAATUALIZACAO = :5, 
			PERMITEREPLICARPRODUTO = :6, 
			USUARIOREPLICOU = :7
		WHERE IDREDE = :8`

	result, err := r.db.ExecContext(ctx, query,
		network.DescricaoRede,
		network.IdRevendedor,
		network.StatusRede,
		network.ReplicarProduto,
		network.DataAtualizacao,
		network.PermiteReplicarProduto,
		network.UsuarioReplicou,
		network.IdRede,
	)

	if err != nil {
		log.Printf("Erro ao atualizar rede %d: %v", network.IdRede, err)
		return fmt.Errorf("erro ao atualizar rede: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Erro ao verificar linhas afetadas: %v", err)
		return fmt.Errorf("erro ao verificar atualização: %w", err)
	}

	if rowsAffected == 0 {
		log.Printf("Nenhuma rede foi atualizada para ID: %d", network.IdRede)
		return fmt.Errorf("rede não encontrada para atualização: %d", network.IdRede)
	}

	log.Printf("Rede %d atualizada com sucesso", network.IdRede)
	return nil
}

// GetNetworkReplicados retrieves all replicated products
func (r *NetworkRepositoryImpl) GetNetworkReplicados() ([]entities.ProductReplicate, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		SELECT DISTINCT IdRevendedor, IdProduto 
		FROM ProdutosReplicados`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Erro ao consultar produtos replicados: %v", err)
		return nil, fmt.Errorf("erro ao consultar produtos replicados: %w", err)
	}
	defer rows.Close()

	var products []entities.ProductReplicate
	for rows.Next() {
		var product entities.ProductReplicate
		err := rows.Scan(&product.IdRevendedor, &product.IdProduto)
		if err != nil {
			log.Printf("Erro ao escanear produto replicado: %v", err)
			continue
		}
		products = append(products, product)
	}

	log.Printf("Encontrados %d produtos replicados", len(products))
	return products, nil
}

// ReplicateProductNetworkSP executes the stored procedure to replicate products
func (r *NetworkRepositoryImpl) ReplicateProductNetworkSP(idNetwork int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `BEGIN sp_ReplicarProdutoRede(:1); END;`

	_, err := r.db.ExecContext(ctx, query, idNetwork)
	if err != nil {
		log.Printf("Erro ao executar sp_ReplicarProdutoRede: %v", err)
		return fmt.Errorf("erro ao replicar produtos da rede: %w", err)
	}

	log.Printf("Produtos replicados com sucesso para rede: %d", idNetwork)
	return nil
}

// RequestReplicateProducts requests product replication for a network
func (r *NetworkRepositoryImpl) RequestReplicateProducts(idNetwork int, userLogin string) (*entities.Success, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := &entities.Success{Message: "", Success: false}

	// First, check if network exists by ID
	var network entities.Network
	checkQuery := `SELECT IDREDE FROM REDE WHERE IDREDE = :1`
	err := r.db.QueryRowContext(ctx, checkQuery, idNetwork).Scan(&network.IdRede)
	if err != nil {
		if err == sql.ErrNoRows {
			result.Success = false
			result.Message = "Rede informada não foi localizada!"
			return result, nil
		}
		result.Message = "Erro ao consultar rede"
		return result, err
	}

	// Set default user if empty
	usuarioReplicou := userLogin
	if usuarioReplicou == "" {
		usuarioReplicou = "System"
	}

	// Update network to request replication
	query := `
		UPDATE REDE SET 
			USUARIOREPLICOU = :1, 
			REPLICARPRODUTO = '1' 
		WHERE IDREDE = :2`

	execResult, err := r.db.ExecContext(ctx, query, usuarioReplicou, network.IdRede)
	if err != nil {
		log.Printf("Erro ao solicitar replicação de produtos: %v", err)
		result.Message = "Falha de execução: requestReplicateProducts"
		return result, err
	}

	rowsAffected, err := execResult.RowsAffected()
	if err != nil {
		log.Printf("Erro ao verificar linhas afetadas: %v", err)
		result.Message = "Falha de execução: requestReplicateProducts"
		return result, err
	}

	if rowsAffected > 0 {
		result.Success = true
		result.Message = "Requisição efetuada com sucesso!"
		log.Printf("Solicitação de replicação criada com sucesso para rede %d por usuário %s", network.IdRede, usuarioReplicou)
	} else {
		result.Success = false
		result.Message = "Falha de execução: requestReplicateProducts"
	}

	return result, nil
}

// MoveIntegrationMarketingStructure moves staging marketing structure data
func (r *NetworkRepositoryImpl) MoveIntegrationMarketingStructure(dataCorte time.Time) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `BEGIN sp_MoverStagingEstruturaMercadologica(:1); END;`

	_, err := r.db.ExecContext(ctx, query, dataCorte)
	if err != nil {
		log.Printf("Erro ao executar sp_MoverStagingEstruturaMercadologica: %v", err)
		return fmt.Errorf("erro ao mover dados de staging da estrutura mercadológica: %w", err)
	}

	log.Printf("Dados de estrutura mercadológica movidos com sucesso para data de corte: %v", dataCorte)
	return nil
}
