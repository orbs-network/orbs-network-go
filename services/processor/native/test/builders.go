// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"fmt"
	sdkContext "github.com/orbs-network/orbs-contract-sdk/go/context"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

// process call

type processCall struct {
	input *services.ProcessCallInput
}

var EXAMPLE_CONTEXT_ID = []byte{0x17, 0x18}

func processCallInput() *processCall {
	p := &processCall{
		input: &services.ProcessCallInput{
			ContextId:              EXAMPLE_CONTEXT_ID,
			ContractName:           "BenchmarkContract",
			MethodName:             "add",
			InputArgumentArray:     (&protocol.ArgumentArrayBuilder{}).Build(),
			AccessScope:            protocol.ACCESS_SCOPE_READ_ONLY,
			CallingPermissionScope: protocol.PERMISSION_SCOPE_SERVICE,
		},
	}
	return p
}

func (p *processCall) Build() *services.ProcessCallInput {
	return p.input
}

func (p *processCall) WithContextId(contextId sdkContext.ContextId) *processCall {
	p.input.ContextId = primitives.ExecutionContextId(contextId)
	return p
}

func (p *processCall) WithUnknownContract() *processCall {
	p.input.ContractName = "UnknownContract"
	p.input.MethodName = "unknownMethod"
	return p
}

func (p *processCall) WithUnknownMethod() *processCall {
	p.input.ContractName = "BenchmarkContract"
	p.input.MethodName = "unknownMethod"
	return p
}

func (p *processCall) WithDeployableCounterContract(counterStart uint64) *processCall {
	p.input.ContractName = primitives.ContractName(fmt.Sprintf("CounterFrom%d", counterStart))
	p.input.MethodName = "start"
	return p
}

func (p *processCall) WithMethod(contractName primitives.ContractName, methodName primitives.MethodName) *processCall {
	p.input.ContractName = contractName
	p.input.MethodName = methodName
	return p
}

func (p *processCall) WithInternalMethod() *processCall {
	p.input.ContractName = "BenchmarkContract"
	p.input.MethodName = "nop"
	return p
}

func (p *processCall) WithPublicMethod() *processCall {
	p.input.ContractName = "BenchmarkContract"
	p.input.MethodName = "add"
	return p.WithArgs(uint64(1), uint64(2))
}

func (p *processCall) WithSystemMethod() *processCall {
	p.input.ContractName = "BenchmarkContract"
	p.input.MethodName = "_init"
	return p.WithArgs()
}

func (p *processCall) WithSystemPermissions() *processCall {
	p.input.CallingPermissionScope = protocol.PERMISSION_SCOPE_SYSTEM
	return p
}

func (p *processCall) WithArgs(args ...interface{}) *processCall {
	p.input.InputArgumentArray = builders.ArgumentsArray(args...)
	return p
}

// get contract info

type getContractInfo struct {
	input *services.GetContractInfoInput
}

func getContractInfoInput() *getContractInfo {
	p := &getContractInfo{
		input: &services.GetContractInfoInput{
			ContractName: "BenchmarkContract",
		},
	}
	return p
}

func (p *getContractInfo) Build() *services.GetContractInfoInput {
	return p.input
}

func (p *getContractInfo) WithUnknownContract() *getContractInfo {
	p.input.ContractName = "UnknownContract"
	return p
}

func (p *getContractInfo) WithSystemService() *getContractInfo {
	p.input.ContractName = "_Deployments"
	return p
}

func (p *getContractInfo) WithRegularService() *getContractInfo {
	p.input.ContractName = "BenchmarkContract"
	return p
}
