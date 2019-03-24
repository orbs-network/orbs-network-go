// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package ethereum

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/pkg/errors"
)

func ABIPackFunctionInputArguments(abi abi.ABI, functionName string, args []interface{}) ([]byte, error) {
	return abi.Pack(functionName, args...)
}

func ABIUnpackFunctionOutputArguments(abi abi.ABI, out interface{}, functionName string, packedOutput []byte) error {
	return abi.Unpack(out, functionName, packedOutput)
}

// go-ethereum normally only unpacks non-indexed event arguments, this hack is needed to make it unpack everything
// the other option was to duplicate its code and alter it, which we prefer not to do
func ABIUnpackAllEventArguments(abi abi.ABI, out interface{}, eventName string, packedOutput []byte) error {
	eventABI, found := abi.Events[eventName]
	if !found {
		return errors.Errorf("event with name '%s' not found in ABI", eventName)
	}

	return cloneEventABIWithoutIndexed(eventABI).Inputs.Unpack(out, packedOutput)
}

func cloneEventABIWithoutIndexed(eventABI abi.Event) abi.Event {
	clone := eventABI
	clone.Inputs = nil
	for _, argClone := range eventABI.Inputs {
		argClone.Indexed = false // hack: temporarily mark as non-indexed
		clone.Inputs = append(clone.Inputs, argClone)
	}
	return clone
}
