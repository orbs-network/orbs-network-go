package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"sync"
)

type EthereumRpcConnection struct {
	connectorCommon

	config ethereumAdapterConfig
	logger log.BasicLogger

	mu struct {
		sync.Mutex
		client bind.ContractBackend
	}
}

func NewEthereumRpcConnection(config ethereumAdapterConfig, logger log.BasicLogger) *EthereumRpcConnection {
	nc := &EthereumRpcConnection{
		config: config,
		logger: logger,
	}
	nc.getContractCaller = nc.dialIfNeededAndReturnClient
	return nc
}

func (nc *EthereumRpcConnection) dial() error {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	if client, err := ethclient.Dial(nc.config.EthereumEndpoint()); err != nil {
		return err
	} else {
		nc.mu.client = client
	}
	return nil
}

func (nc *EthereumRpcConnection) dialIfNeededAndReturnClient() (bind.ContractBackend, error) {
	if nc.mu.client == nil {
		if err := nc.dial(); err != nil {
			return nil, err
		}
	}
	return nc.mu.client, nil
}
