package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/accounts/abi/bind/backends"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/crypto"
	"math/big"
)

type EthereumSimulator struct{}

func NewEthereumSimulatorConnector() EthereumConnection {
	return &EthereumSimulator{}
}

func (es *EthereumSimulator) Dial(endpoint string) (bind.ContractBackend, error) {
	// Generate a new random account and a funded simulator
	key, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	auth := bind.NewKeyedTransactor(key)

	genesisAllocation := map[common.Address]core.GenesisAccount{
		auth.From: {Balance: big.NewInt(10000000000)},
	}

	sim := backends.NewSimulatedBackend(genesisAllocation, 900000000000)

	return sim, nil
}
