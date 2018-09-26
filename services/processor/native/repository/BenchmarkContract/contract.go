package benchmarkcontract

import (
	"errors"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
)

var CONTRACT = sdk.ContractInfo{
	Name:       "BenchmarkContract",
	Permission: sdk.PERMISSION_SCOPE_SERVICE,
	Methods: map[string]sdk.MethodInfo{
		METHOD_INIT.Name:               METHOD_INIT,
		METHOD_NOP.Name:                METHOD_NOP,
		METHOD_ADD.Name:                METHOD_ADD,
		METHOD_SET.Name:                METHOD_SET,
		METHOD_GET.Name:                METHOD_GET,
		METHOD_ARG_TYPES.Name:          METHOD_ARG_TYPES,
		METHOD_THROW.Name:              METHOD_THROW,
		METHOD_PANIC.Name:              METHOD_PANIC,
		METHOD_INVALID_NO_ERROR.Name:   METHOD_INVALID_NO_ERROR,
		METHOD_INVALID_NO_CONTEXT.Name: METHOD_INVALID_NO_CONTEXT,
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

var METHOD_NOP = sdk.MethodInfo{
	Name:           "nop",
	External:       false,
	Access:         sdk.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).nop,
}

func (c *contract) nop(ctx sdk.Context) error {
	return nil
}

///////////////////////////////////////////////////////////////////////////

var METHOD_ADD = sdk.MethodInfo{
	Name:           "add",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).add,
}

func (c *contract) add(ctx sdk.Context, a uint64, b uint64) (uint64, error) {
	return a + b, nil
}

///////////////////////////////////////////////////////////////////////////

var METHOD_SET = sdk.MethodInfo{
	Name:           "set",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_WRITE,
	Implementation: (*contract).set,
}

func (c *contract) set(ctx sdk.Context, a uint64) error {
	return c.State.WriteUint64ByKey(ctx, "example-key", a)
}

///////////////////////////////////////////////////////////////////////////

var METHOD_GET = sdk.MethodInfo{
	Name:           "get",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).get,
}

func (c *contract) get(ctx sdk.Context) (uint64, error) {
	return c.State.ReadUint64ByKey(ctx, "example-key")
}

///////////////////////////////////////////////////////////////////////////

var METHOD_ARG_TYPES = sdk.MethodInfo{
	Name:           "argTypes",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).argTypes,
}

func (c *contract) argTypes(ctx sdk.Context, a1 uint32, a2 uint64, a3 string, a4 []byte) (uint32, uint64, string, []byte, error) {
	return a1 + 1, a2 + 1, a3 + "1", append(a4, 0x01), nil
}

///////////////////////////////////////////////////////////////////////////

var METHOD_THROW = sdk.MethodInfo{
	Name:           "throw",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).throw,
}

func (c *contract) throw(ctx sdk.Context) error {
	return errors.New("example error returned by contract")
}

///////////////////////////////////////////////////////////////////////////

var METHOD_PANIC = sdk.MethodInfo{
	Name:           "panic",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).panic,
}

func (c *contract) panic(ctx sdk.Context) error {
	panic("example panic thrown by contract")
}

///////////////////////////////////////////////////////////////////////////

var METHOD_INVALID_NO_ERROR = sdk.MethodInfo{
	Name:           "invalidNoError",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).invalidNoError,
}

func (c *contract) invalidNoError(ctx sdk.Context) {
	return
}

///////////////////////////////////////////////////////////////////////////

var METHOD_INVALID_NO_CONTEXT = sdk.MethodInfo{
	Name:           "invalidNoContext",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).invalidNoContext,
}

func (c *contract) invalidNoContext() error {
	return nil
}
