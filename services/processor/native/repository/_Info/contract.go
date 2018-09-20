package info_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
)

var CONTRACT = sdk.ContractInfo{
	Name:       "_Info",
	Permission: sdk.PERMISSION_SCOPE_SYSTEM,
	Methods: map[string]sdk.MethodInfo{
		METHOD_INIT.Name:               METHOD_INIT,
		METHOD_GET_SIGNER_ADDRESS.Name: METHOD_GET_SIGNER_ADDRESS,
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

var METHOD_GET_SIGNER_ADDRESS = sdk.MethodInfo{
	Name:           "getSignerAddress",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).getSignerAddress,
}

func (c *contract) getSignerAddress(ctx sdk.Context) ([]byte, error) {
	return c.Address.GetSignerAddress(ctx)
}
