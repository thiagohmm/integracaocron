package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"log"
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		SELECT ID_REDE, DESCRICAO_REDE, ID_REVENDEDOR, STATUS_REDE, REPLICAR_PRODUTO, 
			   DATA_CADASTRO, DATA_ATUALIZACAO, PERMITE_REPLICAR_PRODUTO, USUARIO_REPLICOU
		FROM REDE 
		WHERE PERMITE_REPLICAR_PRODUTO = '1' 
		  AND STATUS_REDE = '1' 
		  AND REPLICAR_PRODUTO = '1'
		ORDER BY ID_REDE`

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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

// ReplicateProductNetwork replicates products for a network
func (r *NetworkRepositoryImpl) ReplicateProductNetwork(idRede int) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This should be implemented based on your business logic
	// For now, it's a placeholder that logs the operation
	log.Printf("ReplicateProductNetwork called for idRede: %d", idRede)

	// Example placeholder query - replace with actual implementation
	query := `UPDATE PRODUTOS_REDE SET STATUS_REPLICACAO = 'ATIVO' WHERE ID_REDE = :1`

	_, err := r.db.ExecContext(ctx, query, idRede)
	if err != nil {
		log.Printf("Erro ao replicar produtos da rede %d: %v", idRede, err)
		return fmt.Errorf("erro ao replicar produtos da rede: %w", err)
	}

	return nil
}

// GetNetworkReplicadosByDealer gets replicated data by dealer (limited to first row)
func (r *NetworkRepositoryImpl) GetNetworkReplicadosByDealer(idRevendedor int) ([]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `SELECT Cod FROM Produtos WHERE IdRevendedor = :1 AND StatusReplicacao = 'PENDENTE'`

	rows, err := r.db.QueryContext(ctx, query, idRevendedor)
	if err != nil {
		log.Printf("Erro ao consultar produtos para replicação: %v", err)
		return nil, fmt.Errorf("erro ao consultar produtos para replicação: %w", err)
	}
	defer rows.Close()

	var products []entities.ProductSelect
	for rows.Next() {
		var product entities.ProductSelect
		err := rows.Scan(&product.Cod)
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

	query := `SELECT Cod FROM Produtos WHERE IdProduto = :1 AND StatusReplicacao = 'ATIVO'`

	rows, err := r.db.QueryContext(ctx, query, idProduto)
	if err != nil {
		log.Printf("Erro ao consultar produtos por replicação de rede: %v", err)
		return nil, fmt.Errorf("erro ao consultar produtos por replicação: %w", err)
	}
	defer rows.Close()

	var products []entities.ProductSelect
	for rows.Next() {
		var product entities.ProductSelect
		err := rows.Scan(&product.Cod)
		if err != nil {
			log.Printf("Erro ao escanear produto para replicação de rede: %v", err)
			continue
		}
		products = append(products, product)
	}

	log.Printf("Encontrados %d produtos para replicação de rede do produto %d", len(products), idProduto)
	return products, nil
}

// GetNetworkByDealer retrieves a network by dealer ID
func (r *NetworkRepositoryImpl) GetNetworkByDealer(idDealer int) (*entities.Network, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `
		SELECT ID_REDE, DESCRICAO_REDE, ID_REVENDEDOR, STATUS_REDE, REPLICAR_PRODUTO, 
			   DATA_CADASTRO, DATA_ATUALIZACAO, PERMITE_REPLICAR_PRODUTO, USUARIO_REPLICOU
		FROM REDE 
		WHERE ID_REVENDEDOR = :1`

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
			DESCRICAO_REDE = :1, 
			ID_REVENDEDOR = :2, 
			STATUS_REDE = :3, 
			REPLICAR_PRODUTO = :4, 
			DATA_ATUALIZACAO = :5, 
			PERMITE_REPLICAR_PRODUTO = :6, 
			USUARIO_REPLICOU = :7
		WHERE ID_REDE = :8`

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
	checkQuery := `SELECT ID_REDE FROM REDE WHERE ID_REDE = :1`
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
			UsuarioReplicou = :1, 
			replicarProduto = '1' 
		WHERE IdRede = :2`

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
