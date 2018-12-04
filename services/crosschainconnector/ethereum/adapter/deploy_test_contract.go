package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"math/big"
)

type DeployingEthereumConnection interface {
	EthereumConnection
	DeploySimpleStorageContract(auth *bind.TransactOpts, stringValue string) ([]byte, error)
	DeployEmitEvent(auth *bind.TransactOpts, abi abi.ABI) ([]byte, *bind.BoundContract, error)
}


// this is a helper for integration test, not used in production code
func (c *connectorCommon) DeploySimpleStorageContract(auth *bind.TransactOpts, stringValue string) ([]byte, error){
	client, err := c.getContractCaller()
	if err != nil {
		return nil, err
	}

	address, _, _, err := contract.DeploySimpleStorage(auth, client, big.NewInt(42), stringValue)
	return address.Bytes(), err
}

// this is a helper for integration test, not used in production code
func (c *connectorCommon) DeployEmitEvent(auth *bind.TransactOpts, abi abi.ABI) ([]byte, *bind.BoundContract, error){
	client, err := c.getContractCaller()
	if err != nil {
		return nil, nil, err
	}

	address, _, err := contract.DeployEmitEvent(auth, client)

	contract := bind.NewBoundContract(address, abi, client, client, client)

	return address.Bytes(), contract, err
}
