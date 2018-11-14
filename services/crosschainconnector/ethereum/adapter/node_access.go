package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
)

type EthereumNodeConnector struct {
	client bind.ContractBackend
}

func NewEthereumConnection() EthereumConnection {
	return &EthereumNodeConnector{}
}

func (nc *EthereumNodeConnector) Dial(endpoint string) error {
	client, err := ethclient.Dial(endpoint)
	nc.client = client
	return err
}

func (nc *EthereumNodeConnector) GetAuth() *bind.TransactOpts {
	return nil
}

func (nc *EthereumNodeConnector) GetClient() bind.ContractBackend {
	return nc.client
}
