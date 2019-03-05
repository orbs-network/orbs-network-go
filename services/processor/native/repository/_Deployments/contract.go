package deployments_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/service"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_GlobalPreOrder"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Info"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

// helpers for avoiding reliance on strings throughout the system
const CONTRACT_NAME = "_Deployments"
const METHOD_GET_INFO = "getInfo"
const METHOD_GET_CODE = "getCode"
const METHOD_DEPLOY_SERVICE = "deployService"

/////////////////////////////////////////////////////////////////
// contract starts here

var PUBLIC = sdk.Export(getInfo, getCode, deployService)

func getInfo(serviceName string) uint32 {
	if isImplicitlyDeployed(serviceName) {
		return uint32(protocol.PROCESSOR_TYPE_NATIVE)
	}

	processorType := _readProcessor(serviceName)
	if processorType == 0 {
		panic("contract not deployed")
	}
	return processorType
}

func isImplicitlyDeployed(serviceName string) bool {
	switch serviceName {
	case
		CONTRACT_NAME,
		globalpreorder_systemcontract.CONTRACT_NAME,
		info_systemcontract.CONTRACT_NAME:
		return true
	}
	return false
}

func getCode(serviceName string) []byte {
	code := _readCode(serviceName)
	if len(code) == 0 {
		panic("contract code not available")
	}
	return code
}

func deployService(serviceName string, processorType uint32, code []byte) {
	// TODO(https://github.com/orbs-network/orbs-network-go/issues/571): sanitize serviceName

	existingProcessorType := _readProcessor(serviceName)
	if existingProcessorType != 0 {
		panic("contract already deployed")
	}

	_writeProcessor(serviceName, processorType)

	if len(code) != 0 {
		_writeCode(serviceName, code)
	}

	service.CallMethod(serviceName, "_init")
}

func _readProcessor(serviceName string) uint32 {
	return state.ReadUint32([]byte(serviceName + ".Processor"))
}

func _writeProcessor(serviceName string, processorType uint32) {
	state.WriteUint32([]byte(serviceName+".Processor"), processorType)
}

func _readCode(serviceName string) []byte {
	return state.ReadBytes([]byte(serviceName + ".Code"))
}

func _writeCode(serviceName string, code []byte) {
	state.WriteBytes([]byte(serviceName+".Code"), code)
}
