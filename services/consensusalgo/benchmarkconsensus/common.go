package benchmarkconsensus

import (
	"github.com/orbs-network/orbs-network-go/crypto"
	"github.com/orbs-network/orbs-network-go/crypto/logic"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
)

func (s *service) lastCommittedBlockHeight() primitives.BlockHeight {
	if s.lastCommittedBlock == nil {
		return 0
	}
	return s.lastCommittedBlock.TransactionsBlock.Header.BlockHeight()
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

func (s *service) validateBlockConsensus(blockPair *protocol.BlockPairContainer, prevCommittedBlockPair *protocol.BlockPairContainer) error {
	// correct block type
	if !blockPair.TransactionsBlock.BlockProof.IsTypeBenchmarkConsensus() {
		return errors.Errorf("incorrect block proof type: %s", blockPair.TransactionsBlock.BlockProof.Type())
	}
	if !blockPair.ResultsBlock.BlockProof.IsTypeBenchmarkConsensus() {
		return errors.Errorf("incorrect block proof type: %s", blockPair.ResultsBlock.BlockProof.Type())
	}

	// prev block hash ptr (if given)
	if prevCommittedBlockPair != nil {
		prevTxHash := crypto.CalcTransactionsBlockHash(prevCommittedBlockPair)
		if !blockPair.TransactionsBlock.Header.PrevBlockHashPtr().Equal(prevTxHash) {
			return errors.Errorf("transactions prev block hash does not match prev block: %s", prevTxHash)
		}
		prevRxHash := crypto.CalcResultsBlockHash(prevCommittedBlockPair)
		if !blockPair.ResultsBlock.Header.PrevBlockHashPtr().Equal(prevRxHash) {
			return errors.Errorf("results prev block hash does not match prev block: %s", prevRxHash)
		}
	}

	// block proof
	blockProof := blockPair.ResultsBlock.BlockProof.BenchmarkConsensus()
	if !blockProof.Sender().SenderPublicKey().Equal(s.config.ConstantConsensusLeader()) {
		return errors.Errorf("block proof not from leader: %s", blockProof.Sender().SenderPublicKey())
	}
	signedData := s.signedDataForBlockProof(blockPair)
	if !signature.VerifyEd25519(blockProof.Sender().SenderPublicKey(), signedData, blockProof.Sender().Signature()) {
		return errors.Errorf("block proof signature is invalid: %s", blockProof.Sender().Signature())
	}

	return nil
}

func (s *service) signedDataForBlockProof(blockPair *protocol.BlockPairContainer) []byte {
	txHash := crypto.CalcTransactionsBlockHash(blockPair)
	rxHash := crypto.CalcResultsBlockHash(blockPair)
	xorHash := logic.CalcXor(txHash, rxHash)
	return xorHash
}

func (s *service) handleBlockConsensusFromHandler(blockType protocol.BlockType, blockPair *protocol.BlockPairContainer, prevCommittedBlockPair *protocol.BlockPairContainer) error {
	if blockType != protocol.BLOCK_TYPE_BLOCK_PAIR {
		return errors.Errorf("handler received unsupported block type %s", blockType)
	}
	err := s.validateBlockConsensus(blockPair, prevCommittedBlockPair)
	return err
}
