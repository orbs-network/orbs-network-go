package globalpreorder

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

var CONTRACT = types.ContractInfo{
	Name:       "_GlobalPreOrder",
	Permission: protocol.PERMISSION_SCOPE_SYSTEM,
	Methods: map[primitives.MethodName]types.MethodInfo{
		METHOD_INIT.Name:    METHOD_INIT,
		METHOD_APPROVE.Name: METHOD_APPROVE,
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
	return nil
}

///////////////////////////////////////////////////////////////////////////

var METHOD_APPROVE = types.MethodInfo{
	Name:           "approve",
	External:       true,
	Access:         protocol.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).approve,
}

func (c *contract) approve(ctx types.Context) error {
	// TODO: add subscription check here
	return nil
}
