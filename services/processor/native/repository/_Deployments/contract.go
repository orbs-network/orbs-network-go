package deployments

import (
	"errors"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
)

var CONTRACT = sdk.ContractInfo{
	Name:       "_Deployments",
	Permission: sdk.PERMISSION_SCOPE_SYSTEM,
	Methods: map[string]sdk.MethodInfo{
		METHOD_INIT.Name:           METHOD_INIT,
		METHOD_GET_INFO.Name:       METHOD_GET_INFO,
		METHOD_GET_CODE.Name:       METHOD_GET_CODE,
		METHOD_DEPLOY_SERVICE.Name: METHOD_DEPLOY_SERVICE,
	},
	InitSingleton: newContract,
}

func newContract(base *sdk.BaseContract) sdk.Contract {
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

var METHOD_GET_INFO = sdk.MethodInfo{
	Name:           "getInfo",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).getInfo,
}

func (c *contract) getInfo(ctx sdk.Context, serviceName string) (uint32, error) {
	if serviceName == "_Deployments" { // getInfo on self
		return uint32(sdk.PROCESSOR_TYPE_NATIVE), nil
	}
	processorType, err := c.State.ReadUint32ByKey(ctx, serviceName+".Processor")
	if err == nil && processorType == 0 {
		err = errors.New("contract not deployed")
	}
	return processorType, err
}

///////////////////////////////////////////////////////////////////////////

var METHOD_GET_CODE = sdk.MethodInfo{
	Name:           "getCode",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).getCode,
}

func (c *contract) getCode(ctx sdk.Context, serviceName string) ([]byte, error) {
	code, err := c.State.ReadBytesByKey(ctx, serviceName+".Code")
	if err == nil && len(code) == 0 {
		err = errors.New("contract code not available")
	}
	return code, err
}

///////////////////////////////////////////////////////////////////////////

var METHOD_DEPLOY_SERVICE = sdk.MethodInfo{
	Name:           "deployService",
	External:       true,
	Access:         sdk.ACCESS_SCOPE_READ_WRITE,
	Implementation: (*contract).deployService,
}

func (c *contract) deployService(ctx sdk.Context, serviceName string, processorType uint32, code []byte) error {
	_, err := c.getInfo(ctx, serviceName)
	if err == nil {
		return errors.New("contract already deployed")
	}

	// TODO: sanitize serviceName

	err = c.State.WriteUint32ByKey(ctx, serviceName+".Processor", processorType)
	if err != nil {
		return fmt.Errorf("failed writing Processor key: %s", err.Error())
	}

	if len(code) != 0 {
		err = c.State.WriteBytesByKey(ctx, serviceName+".Code", code)
		if err != nil {
			return fmt.Errorf("failed writing Code key: %s", err.Error())
		}
	}

	_, err = c.Service.CallMethod(ctx, serviceName, "_init")
	if err != nil {
		return errors.New("failed to initialize contract")
	}

	return nil
}
