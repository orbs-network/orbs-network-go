package test

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type processCall struct {
	input *services.ProcessCallInput
}

func processCallInput() *processCall {
	p := &processCall{
		input: &services.ProcessCallInput{
			ContextId:         0,
			ContractName:      "BenchmarkContract",
			MethodName:        "add",
			InputArguments:    nil,
			AccessScope:       protocol.ACCESS_SCOPE_READ_ONLY,
			PermissionScope:   protocol.PERMISSION_SCOPE_SERVICE,
			CallingService:    "",
			TransactionSigner: nil,
		},
	}
	return p
}

func (p *processCall) Build() *services.ProcessCallInput {
	if p.input.CallingService == "" {
		p.WithSameCallingService()
	}
	return p.input
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

func (p *processCall) WithMethod(contractName primitives.ContractName, methodName primitives.MethodName) *processCall {
	p.input.ContractName = contractName
	p.input.MethodName = methodName
	return p
}

func (p *processCall) WithInternalMethod() *processCall {
	p.input.ContractName = "BenchmarkContract"
	p.input.MethodName = "_init"
	return p
}

func (p *processCall) WithExternalMethod() *processCall {
	p.input.ContractName = "BenchmarkContract"
	p.input.MethodName = "add"
	return p
}

func (p *processCall) WithExternalWriteMethod() *processCall {
	p.input.ContractName = "BenchmarkContract"
	p.input.MethodName = "set"
	return p
}

func (p *processCall) WithSameCallingService() *processCall {
	p.input.CallingService = p.input.ContractName
	return p
}

func (p *processCall) WithDifferentCallingService() *processCall {
	p.input.CallingService = "DifferentFrom" + p.input.ContractName
	return p
}

func (p *processCall) WithSystemPermissions() *processCall {
	p.input.PermissionScope = protocol.PERMISSION_SCOPE_SYSTEM
	return p
}

func (p *processCall) WithWriteAccess() *processCall {
	p.input.AccessScope = protocol.ACCESS_SCOPE_READ_WRITE
	return p
}

func (p *processCall) WithArgs(args []*protocol.MethodArgument) *processCall {
	p.input.InputArguments = args
	return p
}
