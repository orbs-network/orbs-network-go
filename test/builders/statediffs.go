package builders

import (
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

type sdiff struct {
	builder *protocol.ContractStateDiffBuilder
}

func ContractStateDiff() *sdiff {
	return &sdiff{
		builder: &protocol.ContractStateDiffBuilder{
			ContractName: "BenchmarkToken",
			StateDiffs: []*protocol.StateRecordBuilder{
				{Key: []byte("amount"), Value: []byte{10}},
			},
		},
	}
}

func (s *sdiff) Build() *protocol.ContractStateDiff {
	return s.builder.Build()
}

func (s *sdiff) Builder() *protocol.ContractStateDiffBuilder {
	return s.builder
}
