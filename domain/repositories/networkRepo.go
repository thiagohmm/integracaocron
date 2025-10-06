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

// GetNetwork retrieves all networks
func (r *NetworkRepositoryImpl) GetNetwork() ([]entities.Network, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `SELECT ID_REDE, ID_REVENDEDOR FROM REDES ORDER BY ID_REDE`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		log.Printf("Erro ao consultar redes: %v", err)
		return nil, fmt.Errorf("erro ao consultar redes: %w", err)
	}
	defer rows.Close()

	var networks []entities.Network
	for rows.Next() {
		var network entities.Network
		err := rows.Scan(&network.IdRede, &network.IdRevendedor)
		if err != nil {
			log.Printf("Erro ao escanear rede: %v", err)
			continue
		}
		networks = append(networks, network)
	}

	log.Printf("Encontradas %d redes", len(networks))
	return networks, nil
}

// ListByAllByIdDealerNew retrieves dealers by ID
func (r *NetworkRepositoryImpl) ListByAllByIdDealerNew(idRevendedor int) ([]entities.DealerNetwork, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `SELECT ID_REVENDEDOR FROM REVENDEDOR_REDE WHERE ID_REVENDEDOR_PRINCIPAL = :1`

	rows, err := r.db.QueryContext(ctx, query, idRevendedor)
	if err != nil {
		log.Printf("Erro ao consultar revendedores da rede: %v", err)
		return nil, fmt.Errorf("erro ao consultar revendedores: %w", err)
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

	log.Printf("Encontrados %d revendedores para a rede %d", len(dealers), idRevendedor)
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

// GetNetworkReplicadosByDealer gets replicated data by dealer
func (r *NetworkRepositoryImpl) GetNetworkReplicadosByDealer(idRevendedor int) ([]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// This is a placeholder implementation
	// Replace with your actual query and data structure
	log.Printf("GetNetworkReplicadosByDealer called for idRevendedor: %d", idRevendedor)

	query := `SELECT ID_PRODUTO FROM PRODUTOS_REPLICADOS WHERE ID_REVENDEDOR = :1`

	rows, err := r.db.QueryContext(ctx, query, idRevendedor)
	if err != nil {
		log.Printf("Erro ao consultar produtos replicados: %v", err)
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
			"IDPRODUTO": idProduto,
		}
		results = append(results, result)
	}

	return results, nil
}

// GetProductsByReplicateNetworkServiceNew gets products for replication
func (r *NetworkRepositoryImpl) GetProductsByReplicateNetworkServiceNew(idRevendedor int) ([]entities.ProductSelect, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	query := `SELECT COD FROM PRODUTOS WHERE ID_REVENDEDOR = :1 AND STATUS_REPLICACAO = 'PENDENTE'`

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
