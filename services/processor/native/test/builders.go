package test

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
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
			InputArguments:    []*protocol.MethodArgument{},
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
	return p.WithArgs(uint64(1), uint64(2))
}

func (p *processCall) WithExternalWriteMethod() *processCall {
	p.input.ContractName = "BenchmarkContract"
	p.input.MethodName = "set"
	return p.WithArgs(uint64(3))
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

func (p *processCall) WithArgs(args ...interface{}) *processCall {
	p.input.InputArguments = argumentBuilder(args...)
	return p
}

func argumentBuilder(args ...interface{}) (res []*protocol.MethodArgument) {
	res = []*protocol.MethodArgument{}
	for _, arg := range args {
		switch arg.(type) {
		case uint32:
			res = append(res, (&protocol.MethodArgumentBuilder{Name: "uint32", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_32_VALUE, Uint32Value: arg.(uint32)}).Build())
		case uint64:
			res = append(res, (&protocol.MethodArgumentBuilder{Name: "uint64", Type: protocol.METHOD_ARGUMENT_TYPE_UINT_64_VALUE, Uint64Value: arg.(uint64)}).Build())
		case string:
			res = append(res, (&protocol.MethodArgumentBuilder{Name: "string", Type: protocol.METHOD_ARGUMENT_TYPE_STRING_VALUE, StringValue: arg.(string)}).Build())
		case []byte:
			res = append(res, (&protocol.MethodArgumentBuilder{Name: "bytes", Type: protocol.METHOD_ARGUMENT_TYPE_BYTES_VALUE, BytesValue: arg.([]byte)}).Build())
		}
	}
	return
}

func handleSdkCallInput(contractName primitives.ContractName, methodName primitives.MethodName, args []*protocol.MethodArgument) *handlers.HandleSdkCallInput {
	return &handlers.HandleSdkCallInput{
		ContextId:      0,
		ContractName:   contractName,
		MethodName:     methodName,
		InputArguments: args,
	}
}

func handleSdkCallOutput(args []*protocol.MethodArgument) *handlers.HandleSdkCallOutput {
	return &handlers.HandleSdkCallOutput{
		OutputArguments: args,
	}
}
