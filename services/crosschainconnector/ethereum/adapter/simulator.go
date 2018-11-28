package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"math/big"
)

type EthereumSimulator struct {
	connectorCommon

	auth      *bind.TransactOpts
	simClient *backends.SimulatedBackend
	logger    log.BasicLogger
}

func NewEthereumSimulatorConnection(logger log.BasicLogger) *EthereumSimulator {
	e := &EthereumSimulator{
		logger: logger,
	}

	e.createClientAndInitAccount()

	return e
}

func (es *EthereumSimulator) createClientAndInitAccount() {
	// Generate a new random account and a funded simulator
	key, err := crypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	es.auth = bind.NewKeyedTransactor(key)

	genesisAllocation := map[common.Address]core.GenesisAccount{
		es.auth.From: {Balance: big.NewInt(10000000000)},
	}

	es.simClient = backends.NewSimulatedBackend(genesisAllocation, 900000000000)
	es.getContractCaller = func() (bind.ContractBackend, error) {
		return es.simClient, nil
	}
}

func (es *EthereumSimulator) GetAuth() *bind.TransactOpts {
	// this is used for test code, not protecting this
	return es.auth
}

func (es *EthereumSimulator) Commit() {
	es.simClient.Commit()
}
