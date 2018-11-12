package adapter

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

type EthereumConnection interface {
	Dial(endpoint string) (bind.ContractBackend, error)
}
