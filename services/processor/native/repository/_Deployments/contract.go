package deployments_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/service"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/state"
)

var EXPORTS = sdk.Export(getInfo, getCode, deployService)

func getInfo(serviceName string) uint32 {
	if serviceName == "_Deployments" { // getInfo on self
		return uint32(sdk.PROCESSOR_TYPE_NATIVE)
	}
	processorType := state.ReadUint32ByKey(serviceName + ".Processor")
	if processorType == 0 {
		panic("contract not deployed")
	}
	return processorType
}

func getCode(serviceName string) []byte {
	code := state.ReadBytesByKey(serviceName + ".Code")
	if len(code) == 0 {
		panic("contract code not available")
	}
	return code
}

func deployService(serviceName string, processorType uint32, code []byte) {
	getInfo(serviceName) // will panic if already deployed

	// TODO: sanitize serviceName

	state.WriteUint32ByKey(serviceName+".Processor", processorType)

	if len(code) != 0 {
		state.WriteBytesByKey(serviceName+".Code", code)
	}

	service.CallMethod(serviceName, "_init")
}
