package deployments_systemcontract

import (
	"bytes"
	"fmt"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/address"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/service"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"github.com/orbs-network/orbs-network-go/crypto/encoding"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Elections"
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

var PUBLIC = sdk.Export(getInfo, getCode, deployService, lockNativeDeployment, unlockNativeDeployment)

func getInfo(serviceName string) uint32 {
	if _isImplicitlyDeployed(serviceName) {
		return uint32(protocol.PROCESSOR_TYPE_NATIVE)
	}

	processorType := _readProcessor(serviceName)
	if processorType == 0 {
		panic("contract not deployed")
	}
	return processorType
}

func getCode(serviceName string) []byte {
	code := _readCode(serviceName)
	if len(code) == 0 {
		panic("contract code not available")
	}
	return code
}

func deployService(serviceName string, processorType uint32, code []byte) {
	if processorType == uint32(protocol.PROCESSOR_TYPE_NATIVE) {
		_validateNativeDeploymentLock()
	}

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

func lockNativeDeployment() {
	currentOwner := _readNativeDeploymentOwner()
	if len(currentOwner) == 0 {
		_writeNativeDeploymentOwner(address.GetSignerAddress())
	} else {
		panic(fmt.Sprintf("current owner %s must unlockNativeDeployment first", encoding.EncodeHex(currentOwner)))
	}
}

func unlockNativeDeployment() {
	_validateNativeDeploymentLock()
	_writeNativeDeploymentOwner([]byte{})
}

func _validateNativeDeploymentLock() {
	currentOwner := _readNativeDeploymentOwner()
	if len(currentOwner) != 0 && !bytes.Equal(currentOwner, address.GetSignerAddress()) {
		panic(fmt.Sprintf("native deployment is locked to owner %s", encoding.EncodeHex(currentOwner)))
	}
}

func _isImplicitlyDeployed(serviceName string) bool {
	switch serviceName {
	case
		CONTRACT_NAME,
		info_systemcontract.CONTRACT_NAME,
		elections_systemcontract.CONTRACT_NAME,
		globalpreorder_systemcontract.CONTRACT_NAME:
		return true
	}
	return false
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

func _readNativeDeploymentOwner() []byte {
	return state.ReadBytes([]byte("NativeDeploymentOwner"))
}

func _writeNativeDeploymentOwner(newOwnerAddress []byte) {
	state.WriteBytes([]byte("NativeDeploymentOwner"), newOwnerAddress)
}
