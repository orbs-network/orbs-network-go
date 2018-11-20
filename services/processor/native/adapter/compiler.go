package adapter

import (
	"context"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
)

type Compiler interface {
	Compile(ctx context.Context, code string) (*sdkContext.ContractInfo, error)
}
