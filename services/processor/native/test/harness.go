package test

import (
	"github.com/orbs-network/orbs-network-go/services/processor/native"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type harness struct {
	service services.Processor
}

func newHarness() *harness {
	return &harness{
		service: native.NewNativeProcessor(),
	}
}

// builder

type processCall struct {
	input *services.ProcessCallInput
}

func processCallInput() *processCall {
	p := &processCall{
		input: &services.ProcessCallInput{},
	}
	return p
}

func (p *processCall) Build() *services.ProcessCallInput {
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
