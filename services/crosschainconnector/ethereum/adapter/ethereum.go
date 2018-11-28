package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type ethereumAdapterConfig interface {
	EthereumEndpoint() string
}

type EthereumConnection interface {
	GetClient() (bind.ContractBackend, error)
}
