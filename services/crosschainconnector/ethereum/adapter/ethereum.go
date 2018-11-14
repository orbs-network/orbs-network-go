package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type EthereumConnection interface {
	Dial(endpoint string) error
	GetAuth() *bind.TransactOpts // for simulation usage only
	GetClient() bind.ContractBackend
}
