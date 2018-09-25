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

const TOTAL_SUPPLY = 10000

///////////////////////////////////////////////////////////////////////////

var METHOD_INIT = sdk.MethodInfo{
	Name:           "_init",
	External:       false,
	Access:         sdk.ACCESS_SCOPE_READ_WRITE,
	Implementation: (*contract)._init,
}

func (c *contract) _init(ctx sdk.Context) error {
	ownerAddress, err := c.Address.GetSignerAddress(ctx)
	if err != nil {
		return err
	}
	return c.State.WriteUint64ByAddress(ctx, ownerAddress, TOTAL_SUPPLY)
}

///////////////////////////////////////////////////////////////////////////

var METHOD_TRANSFER = sdk.MethodInfo{
	Name:           "transfer",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_WRITE,
	Implementation: (*contract).transfer,
}

func (c *contract) transfer(ctx sdk.Context, amount uint64, targetAddress []byte) error {
	// sender
	callerAddress, err := c.Address.GetCallerAddress(ctx)
	if err != nil {
		return err
	}
	callerBalance, err := c.State.ReadUint64ByAddress(ctx, callerAddress)
	if err != nil {
		return err
	}
	if callerBalance < amount {
		return fmt.Errorf("transfer of %d failed since balance is only %d", amount, callerBalance)
	}
	err = c.State.WriteUint64ByAddress(ctx, callerAddress, callerBalance-amount)
	if err != nil {
		return err
	}

	// recipient
	err = c.Address.ValidateAddress(ctx, targetAddress)
	if err != nil {
		return err
	}
	targetBalance, err := c.State.ReadUint64ByAddress(ctx, targetAddress)
	if err != nil {
		return err
	}
	return c.State.WriteUint64ByAddress(ctx, targetAddress, targetBalance+amount)
}

///////////////////////////////////////////////////////////////////////////

var METHOD_GET_BALANCE = sdk.MethodInfo{
	Name:           "getBalance",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).getBalance,
}

func (c *contract) getBalance(ctx sdk.Context, targetAddress []byte) (uint64, error) {
	err := c.Address.ValidateAddress(ctx, targetAddress)
	if err != nil {
		return 0, err
	}
	return c.State.ReadUint64ByAddress(ctx, targetAddress)
}
