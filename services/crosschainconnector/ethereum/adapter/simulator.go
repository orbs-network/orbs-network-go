package adapter

import (
	"context"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"math/big"
	"sync"
)

type EthereumSimulator struct {
	auth      *bind.TransactOpts
	simClient bind.ContractBackend
	config    ethereumAdapterConfig
	logger    log.BasicLogger
	mu        sync.Mutex
}

func NewEthereumSimulatorConnection(config ethereumAdapterConfig, logger log.BasicLogger) *EthereumSimulator {
	return &EthereumSimulator{
		config: config,
		logger: logger,
	}
}

func (es *EthereumSimulator) Dial(endpoint string) error {
	es.mu.Lock()
	defer es.mu.Unlock()
	if es.simClient == nil {
		// Generate a new random account and a funded simulator
		key, err := crypto.GenerateKey()
		if err != nil {
			return err
		}
		es.auth = bind.NewKeyedTransactor(key)

		genesisAllocation := map[common.Address]core.GenesisAccount{
			es.auth.From: {Balance: big.NewInt(10000000000)},
		}

		sim := backends.NewSimulatedBackend(genesisAllocation, 900000000000)
		es.simClient = sim
	}
	return nil
}

func (es *EthereumSimulator) GetAuth() *bind.TransactOpts {
	// this is used for test code, not protecting this
	return es.auth
}

func (es *EthereumSimulator) GetClient() (bind.ContractBackend, error) {
	if es.simClient == nil {
		es.logger.Info("connecting to ethereum simulator", log.String("endpoint", es.config.EthereumEndpoint()))
		if err := es.Dial(es.config.EthereumEndpoint()); err != nil {
			return nil, err
		}
	}
	return es.simClient, nil
}


func (es *EthereumSimulator) DeployStorageContract(ctx context.Context, number int64, text string) (string, error) {
	if err := es.Dial(""); err != nil { // create the client so we can deploy
		return "", err
	}
	client, err := es.GetClient()
	if err != nil {
		return "", err
	}
	address, _, _, err := contract.DeploySimpleStorage(es.GetAuth(), client, big.NewInt(number), text)
	if err != nil {
		return "", err
	}
	client.(*backends.SimulatedBackend).Commit() // assuming simulation, this will commit the pending transactions
	return hexutil.Encode(address[:]), nil
}