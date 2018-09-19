package globalpreorder

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
)

var CONTRACT = sdk.ContractInfo{
	Name:       "_GlobalPreOrder",
	Permission: sdk.PERMISSION_SCOPE_SYSTEM,
	Methods: map[string]sdk.MethodInfo{
		METHOD_INIT.Name:    METHOD_INIT,
		METHOD_APPROVE.Name: METHOD_APPROVE,
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

var METHOD_APPROVE = sdk.MethodInfo{
	Name:           "approve",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).approve,
}

func (c *contract) approve(ctx sdk.Context) error {
	// TODO: add subscription check here
	return nil
}
