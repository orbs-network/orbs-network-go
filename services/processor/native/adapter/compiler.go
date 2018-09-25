package adapter

import (
	"context"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
)

type Compiler interface {
	Compile(ctx context.Context, code string) (*sdk.ContractInfo, error)
}
