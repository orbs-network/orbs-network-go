package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type ethereumAdapterConfig interface {
	EthereumEndpoint() string
}

type EthereumConnection interface {
	GetAuth() *bind.TransactOpts // for simulation usage only
	GetClient() (bind.ContractBackend, error)
}
