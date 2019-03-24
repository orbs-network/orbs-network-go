// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package digest

import (
	"github.com/orbs-network/lean-helix-go"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/pkg/errors"
)

func GetBlockSignersFromReceiptProof(packedProof primitives.PackedReceiptProof) ([]primitives.NodeAddress, error) {
	var res []primitives.NodeAddress
	receiptProof := protocol.ReceiptProofReader(packedProof)
	switch receiptProof.BlockProof().Type() {
	case protocol.RESULTS_BLOCK_PROOF_TYPE_LEAN_HELIX:
		leanHelixBlockProof := receiptProof.BlockProof().LeanHelix()
		memberIds, err := leanhelix.GetMemberIdsFromBlockProof(leanHelixBlockProof)
		if err != nil {
			return nil, err
		}
		for _, memberId := range memberIds {
			res = append(res, primitives.NodeAddress(memberId))
		}
		return res, nil
	case protocol.RESULTS_BLOCK_PROOF_TYPE_BENCHMARK_CONSENSUS:
		benchmarkConsensusBlockProof := receiptProof.BlockProof().BenchmarkConsensus()
		iterator := benchmarkConsensusBlockProof.NodesIterator()
		for iterator.HasNext() {
			res = append(res, iterator.NextNodes().SenderNodeAddress())
		}
		return res, nil
	}
	return nil, errors.Errorf("unknown block proof type: %v", receiptProof.BlockProof().Type())
}
