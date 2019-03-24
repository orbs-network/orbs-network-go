// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package builders

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"time"
)

type ethereumCallContractInput struct {
	timestamp       primitives.TimestampNano
	functionName    string
	abi             string
	contractAddress string
	packedArgs      []byte
	blockNumber     uint64
}

func EthereumCallContractInput() *ethereumCallContractInput {
	return &ethereumCallContractInput{
		timestamp:       primitives.TimestampNano(time.Now().UnixNano()),
		functionName:    "placeholder",
		packedArgs:      nil,
		contractAddress: "0xABCDEF",
		abi:             "[]",
		blockNumber:     0,
	}
}

func (ec *ethereumCallContractInput) WithTimestamp(t time.Time) *ethereumCallContractInput {
	ec.timestamp = primitives.TimestampNano(t.UnixNano())
	return ec
}

func (ec *ethereumCallContractInput) WithFunctionName(name string) *ethereumCallContractInput {
	ec.functionName = name
	return ec
}

func (ec *ethereumCallContractInput) WithAbi(json string) *ethereumCallContractInput {
	ec.abi = json
	return ec
}

func (ec *ethereumCallContractInput) WithContractAddress(address string) *ethereumCallContractInput {
	ec.contractAddress = address
	return ec
}

func (ec *ethereumCallContractInput) WithPackedArguments(data []byte) *ethereumCallContractInput {
	ec.packedArgs = data
	return ec
}

func (ec *ethereumCallContractInput) Build() *services.EthereumCallContractInput {
	return &services.EthereumCallContractInput{
		ReferenceTimestamp:              ec.timestamp,
		EthereumAbiPackedInputArguments: ec.packedArgs,
		EthereumContractAddress:         ec.contractAddress,
		EthereumJsonAbi:                 ec.abi,
		EthereumFunctionName:            ec.functionName,
		EthereumBlockNumber:             ec.blockNumber,
	}
}
