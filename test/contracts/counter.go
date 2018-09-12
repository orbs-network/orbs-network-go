package contracts

import "fmt"

const counterContractCode = `
package counter

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

var CONTRACT = types.ContractInfo{
	Name:       "CounterFrom%d",
	Permission: protocol.PERMISSION_SCOPE_SERVICE,
	Methods: map[primitives.MethodName]types.MethodInfo{
		METHOD_INIT.Name: METHOD_INIT,
		METHOD_ADD.Name:  METHOD_ADD,
		METHOD_GET.Name:  METHOD_GET,
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
	return c.State.WriteUint64ByKey(ctx, "count", %d)
}

///////////////////////////////////////////////////////////////////////////

var METHOD_ADD = types.MethodInfo{
	Name:           "add",
	External:       true,
	Access:         protocol.ACCESS_SCOPE_READ_WRITE,
	Implementation: (*contract).add,
}

func (c *contract) add(ctx types.Context, amount uint64) error {
	count, err := c.State.ReadUint64ByKey(ctx, "count")
	if err != nil {
		return err
	}
	count += amount
	return c.State.WriteUint64ByKey(ctx, "count", count)
}

///////////////////////////////////////////////////////////////////////////

var METHOD_GET = types.MethodInfo{
	Name:           "get",
	External:       true,
	Access:         protocol.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).get,
}

func (c *contract) get(ctx types.Context) (uint64, error) {
	return c.State.ReadUint64ByKey(ctx, "count")
}
`

func SourceCodeForCounter(startFrom uint64) string {
	return fmt.Sprintf(counterContractCode, startFrom, startFrom)
}
