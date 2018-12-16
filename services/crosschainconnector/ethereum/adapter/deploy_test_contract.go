package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/contract"
	"math/big"
	"strings"
)

type DeployingEthereumConnection interface {
	EthereumConnection
	DeploySimpleStorageContract(auth *bind.TransactOpts, stringValue string) ([]byte, error)
	DeployEthereumContract(auth *bind.TransactOpts, abijson string, bytecode string, params ...interface{}) (*common.Address, *bind.BoundContract, error)
}

// this is a helper for integration test, not used in production code
func (c *connectorCommon) DeploySimpleStorageContract(auth *bind.TransactOpts, stringValue string) ([]byte, error) {
	client, err := c.getContractCaller()
	if err != nil {
		return nil, err
	}

	address, _, _, err := contract.DeploySimpleStorage(auth, client, big.NewInt(42), stringValue)
	return address.Bytes(), err
}

func (c *connectorCommon) DeployEthereumContract(auth *bind.TransactOpts, abijson string, bytecode string, params ...interface{}) (*common.Address, *bind.BoundContract, error) {
	client, err := c.getContractCaller()
	if err != nil {
		return nil, nil, err
	}

	// deploy
	parsedAbi, err := abi.JSON(strings.NewReader(abijson))
	if err != nil {
		return nil, nil, err
	}
	address, _, contract, err := bind.DeployContract(auth, parsedAbi, common.FromHex(bytecode), client, params...)
	if err != nil {
		return nil, nil, err
	}

	return &address, contract, nil
}
