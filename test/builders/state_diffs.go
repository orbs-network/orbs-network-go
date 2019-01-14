package builders

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

/// Test builders for: protocol.ContractStateDiff

type sdiff struct {
	builder *protocol.ContractStateDiffBuilder
	records []*protocol.StateRecordBuilder
}

func ContractStateDiff() *sdiff {
	return &sdiff{
		builder: &protocol.ContractStateDiffBuilder{
			ContractName: "BenchmarkToken",
		},
	}
}

func (s *sdiff) Build() *protocol.ContractStateDiff {
	if len(s.records) == 0 {
		s.records = append(s.records, &protocol.StateRecordBuilder{Key: []byte("amount"), Value: []byte{10}})
	}

	s.builder.StateDiffs = s.records
	return s.builder.Build()
}

func (s *sdiff) Builder() *protocol.ContractStateDiffBuilder {
	return s.builder
}

func (s *sdiff) WithStringRecord(k string, v string) *sdiff {
	s.records = append(s.records, &protocol.StateRecordBuilder{Key: []byte(k), Value: []byte(v)})
	return s
}

func (s *sdiff) WithContractName(contract string) *sdiff {
	s.builder.ContractName = primitives.ContractName(contract)
	return s
}
