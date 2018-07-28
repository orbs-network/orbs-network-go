package benchmarkconsensus

import (
	"github.com/orbs-network/orbs-network-go/crypto"
	"github.com/orbs-network/orbs-network-go/crypto/logic"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
)

func (s *service) lastCommittedBlockHeight() primitives.BlockHeight {
	if s.lastCommittedBlock == nil {
		return 0
	}
	return s.lastCommittedBlock.TransactionsBlock.Header.BlockHeight()
}

func (s *service) signedDataForBlockProof(blockPair *protocol.BlockPairContainer) []byte {
	txHash := crypto.CalcTransactionsBlockHash(blockPair)
	rxHash := crypto.CalcResultsBlockHash(blockPair)
	xorHash := logic.CalcXor(txHash, rxHash)
	return xorHash
}
