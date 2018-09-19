package adapter

import "github.com/orbs-network/orbs-contract-sdk/go/sdk"

type Compiler interface {
	Compile(code string) (*sdk.ContractInfo, error)
}