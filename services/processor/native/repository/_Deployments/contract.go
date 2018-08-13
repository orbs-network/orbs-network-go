package deployments

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native/types"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

var CONTRACT = types.ContractInfo{
	Name:       "_Deployments",
	Permission: protocol.PERMISSION_SCOPE_SYSTEM,
	Methods: []types.MethodInfo{
		METHOD_INIT,
		METHOD_IS_SERVICE_DEPLOYED,
		METHOD_LOAD,
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

var METHOD_IS_SERVICE_DEPLOYED = types.MethodInfo{
	Name:           "isServiceDeployed",
	External:       true,
	Access:         protocol.ACCESS_SCOPE_READ_ONLY,
	Implementation: (*contract).isServiceDeployed,
}

func (c *contract) isServiceDeployed(ctx types.Context, serviceName string) (uint32, error) {
	return c.State.ReadUint32ByKey(ctx, serviceName+".Processor")
}

///////////////////////////////////////////////////////////////////////////

var METHOD_LOAD = types.MethodInfo{
	Name:           "loadService",
	External:       true,
	Access:         protocol.ACCESS_SCOPE_READ_WRITE,
	Implementation: (*contract).loadService,
}

func (c *contract) loadService(ctx types.Context, serviceName string) (uint32, error) {
	processorType, err := c.isServiceDeployed(ctx, serviceName)
	if err == nil {
		return processorType, nil
	}

	// try to deploy if native service

	err = c.Service.IsNative(ctx, serviceName)
	if err != nil {
		return 0, errors.New("unknown service")
	}

	err = c.Service.CallMethod(ctx, serviceName, "_init")
	if err != nil {
		return 0, errors.New("failed to initialize native service")
	}

	err = c.State.WriteUint32ByKey(ctx, serviceName+".Processor", uint32(protocol.PROCESSOR_TYPE_NATIVE))
	if err != nil {
		return 0, errors.Wrap(err, "failed writing Processor key")
	}

	return uint32(protocol.PROCESSOR_TYPE_NATIVE), nil
}
