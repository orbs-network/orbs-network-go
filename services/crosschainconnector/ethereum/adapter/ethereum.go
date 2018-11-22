package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type ethereumAdapterConfig interface {
	EthereumEndpoint() string
}

type EthereumConnection interface {
	Dial(endpoint string) error
	GetAuth() *bind.TransactOpts // for simulation usage only
	GetClient() (bind.ContractBackend, error)
}
