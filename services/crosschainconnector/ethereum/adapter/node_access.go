package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/ethclient"
)

type EthereumNodeConnector struct{}

func NewEthereumConnection() EthereumConnection {
	return &EthereumNodeConnector{}
}

func (nc EthereumNodeConnector) Dial(endpoint string) (bind.ContractBackend, error) {
	client, err := ethclient.Dial(endpoint)
	return client, err
}
