package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"sync"
)

type EthereumNodeAdapter struct {
	client bind.ContractBackend
	config ethereumAdapterConfig
	logger log.BasicLogger
	mu     sync.Mutex
}

func NewEthereumConnection(config ethereumAdapterConfig, logger log.BasicLogger) EthereumConnection {
	return &EthereumNodeAdapter{
		config: config,
		logger: logger,
	}
}

func (nc *EthereumNodeAdapter) dial(endpoint string) error {
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

func (nc *EthereumNodeAdapter) GetClient() (bind.ContractBackend, error) {
	if nc.client == nil {
		nc.logger.Info("connecting to ethereum", log.String("endpoint", nc.config.EthereumEndpoint()))
		if err := nc.dial(nc.config.EthereumEndpoint()); err != nil {
			return nil, err
		}
	}
	return nc.client, nil
}
