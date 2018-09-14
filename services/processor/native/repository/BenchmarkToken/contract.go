package benchmarktoken

import (
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
)

var CONTRACT = sdk.ContractInfo{
	Name:       "BenchmarkToken",
	Permission: sdk.PERMISSION_SCOPE_SERVICE,
	Methods: map[string]sdk.MethodInfo{
		METHOD_INIT.Name:        METHOD_INIT,
		METHOD_TRANSFER.Name:    METHOD_TRANSFER,
		METHOD_GET_BALANCE.Name: METHOD_GET_BALANCE,
	},
	InitSingleton: newContract,
}

func newContract(base *sdk.BaseContract) sdk.ContractInstance {
	return &contract{base}
}

type contract struct{ *sdk.BaseContract }

///////////////////////////////////////////////////////////////////////////

var METHOD_INIT = sdk.MethodInfo{
	Name:           "_init",
	External:       false,
	Access:         sdk.ACCESS_SCOPE_READ_WRITE,
	Implementation: (*contract)._init,
}

func (c *contract) _init(ctx sdk.Context) error {
	return nil
}

///////////////////////////////////////////////////////////////////////////

var METHOD_TRANSFER = sdk.MethodInfo{
	Name:           "transfer",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_WRITE,
	Implementation: (*contract).transfer,
}

func (c *contract) transfer(ctx sdk.Context, amount uint64) error {
	if amount > 1000 {
		return fmt.Errorf("cannot transfer amounts above 1000: %d", amount)
	}
	balance, err := c.State.ReadUint64ByKey(ctx, "total-balance")
	if err != nil {
		return err
	}
	balance += amount
	return c.State.WriteUint64ByKey(ctx, "total-balance", balance)
}

///////////////////////////////////////////////////////////////////////////

var METHOD_GET_BALANCE = sdk.MethodInfo{
	Name:           "getBalance",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).getBalance,
}

func (c *contract) getBalance(ctx sdk.Context) (uint64, error) {
	return c.State.ReadUint64ByKey(ctx, "total-balance")
}
