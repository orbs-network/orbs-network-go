// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package deployments_systemcontract

import (
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/service"
	"github.com/orbs-network/orbs-contract-sdk/go/sdk/v1/state"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/Committee"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/GlobalPreOrder"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Elections"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Info"
	"github.com/orbs-network/orbs-network-go/services/processor/native/repository/_Triggers"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"strconv"
)

func getInfo(serviceName string) uint32 {
	if IsImplicitlyDeployed(serviceName) {
		return uint32(protocol.PROCESSOR_TYPE_NATIVE)
	}

	processorType := _readProcessor(serviceName)
	if processorType == 0 {
		panic("contract not deployed")
	}
	return processorType
}

func getCode(serviceName string) []byte {
	return getCodePart(serviceName, 0)
}

func getCodePart(serviceName string, index uint32) []byte {
	code := _readCode(serviceName, index)
	if len(code) == 0 {
		panic("contract code not available")
	}

	return code
}

func getCodeParts(serviceName string) uint32 {
	processorType := _readProcessor(serviceName)
	if processorType == 0 {
		panic("contract not deployed")
	}
	return _codeCounter(serviceName) + 1
}

func deployService(serviceName string, processorType uint32, code ...[]byte) {
	if processorType == uint32(protocol.PROCESSOR_TYPE_NATIVE) {
		_validateNativeDeploymentLock()
	}

	_validateServiceName(serviceName)

	_addServiceName(serviceName)

	// this read is for backwards compatibility if someone deployed a contract before service name sanitization was added
	existingProcessorType := _readProcessor(serviceName)
	if existingProcessorType != 0 {
		panic("contract already deployed")
	}

	_writeProcessor(serviceName, processorType)

	if len(code) > 0 {
		for i, c := range code {
			_writeCode(serviceName, c, uint32(i))
		}
	} else {
		panic("contract doesn't have any code")
	}

	if existingProcessorType == 1 {
		service.CallMethod(serviceName, "_init")
	}
}

// Function was made go "public" to allow testing, it is not public in the contract.
func IsImplicitlyDeployed(serviceName string) bool {
	switch serviceName {
	case
		CONTRACT_NAME,
		info_systemcontract.CONTRACT_NAME,
		triggers_systemcontract.CONTRACT_NAME,
		committee_systemcontract.CONTRACT_NAME,
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

func _readCode(serviceName string, index uint32) []byte {
	return state.ReadBytes(_codeKey(serviceName, index))
}

func _writeCode(serviceName string, code []byte, index uint32) {
	state.WriteBytes(_codeKey(serviceName, index), code)
	if index > 0 { // backwards compatibility
		_codeCounterIncrement(serviceName)
	}
}

func _codeKey(serviceName string, index uint32) []byte {
	if index == 0 { // backwards compatibility
		return []byte(serviceName + ".Code")
	}
	return []byte(serviceName + ".Code." + strconv.FormatInt(int64(index), 10))
}

func _codeCounter(serviceName string) uint32 {
	return state.ReadUint32(_codeCounterKey(serviceName))
}

func _codeCounterIncrement(serviceName string) {
	counter := _codeCounter(serviceName)
	state.WriteUint32(_codeCounterKey(serviceName), counter+1)
}

func _codeCounterKey(serviceName string) []byte {
	return []byte(serviceName + ".CodeParts")
}
