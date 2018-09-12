package deployments

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

var CONTRACT = types.ContractInfo{
	Name:       "_Deployments",
	Permission: protocol.PERMISSION_SCOPE_SYSTEM,
	Methods: map[primitives.MethodName]types.MethodInfo{
		METHOD_INIT.Name:           METHOD_INIT,
		METHOD_GET_INFO.Name:       METHOD_GET_INFO,
		METHOD_GET_CODE.Name:       METHOD_GET_CODE,
		METHOD_DEPLOY_SERVICE.Name: METHOD_DEPLOY_SERVICE,
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

var METHOD_GET_INFO = types.MethodInfo{
	Name:           "getInfo",
	External:       true,
	Access:         protocol.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).getInfo,
}

func (c *contract) getInfo(ctx types.Context, serviceName string) (uint32, error) {
	if serviceName == "_Deployments" { // getInfo on self
		return uint32(protocol.PROCESSOR_TYPE_NATIVE), nil
	}
	processorType, err := c.State.ReadUint32ByKey(ctx, serviceName+".Processor")
	if err == nil && processorType == 0 {
		err = errors.New("contract not deployed")
	}
	return processorType, err
}

///////////////////////////////////////////////////////////////////////////

var METHOD_GET_CODE = types.MethodInfo{
	Name:           "getCode",
	External:       true,
	Access:         protocol.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).getCode,
}

func (c *contract) getCode(ctx types.Context, serviceName string) ([]byte, error) {
	code, err := c.State.ReadBytesByKey(ctx, serviceName+".Code")
	if err == nil && len(code) == 0 {
		err = errors.New("contract code not available")
	}
	return code, err
}

///////////////////////////////////////////////////////////////////////////

var METHOD_DEPLOY_SERVICE = types.MethodInfo{
	Name:           "deployService",
	External:       true,
	Access:         protocol.ACCESS_SCOPE_READ_WRITE,
	Implementation: (*contract).deployService,
}

func (c *contract) deployService(ctx types.Context, serviceName string, processorType uint32, code []byte) error {
	_, err := c.getInfo(ctx, serviceName)
	if err == nil {
		return errors.New("contract already deployed")
	}

	err = c.State.WriteUint32ByKey(ctx, serviceName+".Processor", processorType)
	if err != nil {
		return errors.Wrap(err, "failed writing Processor key")
	}

	if len(code) != 0 {
		err = c.State.WriteBytesByKey(ctx, serviceName+".Code", code)
		if err != nil {
			return errors.Wrap(err, "failed writing Code key")
		}
	}

	_, err = c.Service.CallMethod(ctx, serviceName, "_init")
	if err != nil {
		errors.New("failed to initialize contract")
	}

	return nil
}
