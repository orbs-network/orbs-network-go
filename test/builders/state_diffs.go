// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

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
