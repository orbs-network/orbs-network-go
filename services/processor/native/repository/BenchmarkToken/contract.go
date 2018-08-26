package benchmarktoken

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

var CONTRACT = types.ContractInfo{
	Name:       "BenchmarkToken",
	Permission: protocol.PERMISSION_SCOPE_SERVICE,
	Methods: []types.MethodInfo{
		METHOD_INIT,
		METHOD_TRANSFER,
		METHOD_GET_BALANCE,
	},
	InitSingleton: newContract,
}

func newContract(base *types.BaseContract) types.Contract {
	return &contract{base}
}

type contract struct{ *types.BaseContract }

///////////////////////////////////////////////////////////////////////////

var METHOD_INIT = types.MethodInfo{
	Name:           "_init",
	External:       false,
	Access:         protocol.ACCESS_SCOPE_READ_WRITE,
	Implementation: (*contract)._init,
}

func (c *contract) _init(ctx types.Context) error {
	return c.State.WriteUint64ByKey(ctx, "total-balance", 0)
}

///////////////////////////////////////////////////////////////////////////

var METHOD_TRANSFER = types.MethodInfo{
	Name:           "transfer",
	External:       true,
	Access:         protocol.ACCESS_SCOPE_READ_WRITE,
	Implementation: (*contract).transfer,
}

func (c *contract) transfer(ctx types.Context, amount uint64) error {
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

var METHOD_GET_BALANCE = types.MethodInfo{
	Name:           "getBalance",
	External:       true,
	Access:         protocol.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).getBalance,
}

func (c *contract) getBalance(ctx types.Context) (uint64, error) {
	return c.State.ReadUint64ByKey(ctx, "total-balance")
}
