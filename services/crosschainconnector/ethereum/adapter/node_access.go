package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"sync"
)

type EthereumNodeAdapter struct {
	client bind.ContractBackend
	mu     sync.Mutex
}

func NewEthereumConnection() EthereumConnection {
	return &EthereumNodeAdapter{}
}

func (nc *EthereumNodeAdapter) Dial(endpoint string) error {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	if nc.client == nil {
		if client, err := ethclient.Dial(endpoint); err != nil {
			return err
		} else {
			nc.client = client
		}
	}
	return nil
}

func (nc *EthereumNodeAdapter) GetAuth() *bind.TransactOpts {
	return nil
}

func (nc *EthereumNodeAdapter) GetClient() bind.ContractBackend {
	return nc.client
}
