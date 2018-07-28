package benchmarkconsensus

import (
	"github.com/orbs-network/orbs-network-go/crypto"
	"github.com/orbs-network/orbs-network-go/crypto/logic"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
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

func (s *service) saveToBlockStorage(blockPair *protocol.BlockPairContainer) error {
	if blockPair.TransactionsBlock.Header.BlockHeight() == 0 {
		return nil
	}
	s.reporting.Infof("Saving block %d to storage", blockPair.TransactionsBlock.Header.BlockHeight())
	_, err := s.blockStorage.CommitBlock(&services.CommitBlockInput{
		BlockPair: blockPair,
	})
	if err != nil {
		return err
	}
	return nil
}
