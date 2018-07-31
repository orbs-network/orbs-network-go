package benchmarkcontract

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

var CONTRACT = types.ContractInfo{
	Name:       "BenchmarkContract",
	Permission: protocol.PERMISSION_SCOPE_SERVICE,
	Methods: []types.MethodInfo{
		METHOD_INIT,
		METHOD_ADD,
	},
	Context: NewContext,
}

func NewContext(base *types.BaseContext) types.Context {
	return &contract{base}
}

type contract struct{ *types.BaseContext }

///////////////////////////////////////////////////////////////////////////

var METHOD_INIT = types.MethodInfo{
	Name:           "_init",
	External:       false,
	Access:         protocol.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract)._init,
}

func (c *contract) _init() error {
	return nil
}

///////////////////////////////////////////////////////////////////////////

var METHOD_ADD = types.MethodInfo{
	Name:           "add",
	External:       true,
	Access:         protocol.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).add,
}

func (c *contract) add(a uint64, b uint64) (uint64, error) {
	c.SdkCall()
	return a + b, nil
}
