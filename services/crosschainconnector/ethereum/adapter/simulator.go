package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
	"sync"
)

type EthereumSimulator struct {
	auth      *bind.TransactOpts
	simClient bind.ContractBackend
	mu        sync.Mutex
}

func NewEthereumSimulatorConnector() EthereumConnection {
	return &EthereumSimulator{}
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
	return es.auth
}

func (es *EthereumSimulator) GetClient() bind.ContractBackend {
	return es.simClient
}
