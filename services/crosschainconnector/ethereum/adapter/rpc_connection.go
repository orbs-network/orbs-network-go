package adapter

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"sync"
)

type EthereumRpcConnection struct {
	connectorCommon

	config ethereumAdapterConfig

	mu struct {
		sync.Mutex
		client EthereumCaller
	}
}

func NewEthereumRpcConnection(config ethereumAdapterConfig, logger log.BasicLogger) *EthereumRpcConnection {
	rpc := &EthereumRpcConnection{
		config: config,
	}
	rpc.logger = logger.WithTags(log.String("adapter", "ethereum"))
	rpc.getContractCaller = rpc.dialIfNeededAndReturnClient
	return rpc
}

func (rpc *EthereumRpcConnection) dial() error {
	rpc.mu.Lock()
	defer rpc.mu.Unlock()
	if client, err := ethclient.Dial(rpc.config.EthereumEndpoint()); err != nil {
		return err
	} else {
		rpc.mu.client = client
	}
	return nil
}

func (rpc *EthereumRpcConnection) dialIfNeededAndReturnClient() (EthereumCaller, error) {
	if rpc.mu.client == nil {
		if err := rpc.dial(); err != nil {
			return nil, err
		}
	}
	return rpc.mu.client, nil
}
