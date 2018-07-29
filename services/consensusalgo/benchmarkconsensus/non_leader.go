package benchmarkconsensus

import (
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
)

func (s *service) nonLeaderHandleCommit(blockPair *protocol.BlockPairContainer) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	err := s.nonLeaderValidateBlockUnsafe(blockPair)
	if err != nil {
		s.reporting.Error(err) // TODO: wrap with added context
		return
	}
	err = s.nonLeaderCommitAndReplyUnsafe(blockPair)
	if err != nil {
		s.reporting.Error(err) // TODO: wrap with added context
		return
	}
}

func (s *service) nonLeaderValidateBlockUnsafe(blockPair *protocol.BlockPairContainer) error {
	// nils
	if blockPair.TransactionsBlock == nil ||
		blockPair.ResultsBlock == nil ||
		blockPair.TransactionsBlock.Header == nil ||
		blockPair.ResultsBlock.Header == nil ||
		blockPair.ResultsBlock.BlockProof == nil {
		panic("invalid block: missing fields")
	}

	// block height
	blockHeight := blockPair.TransactionsBlock.Header.BlockHeight()
	if blockHeight != blockPair.ResultsBlock.Header.BlockHeight() {
		return errors.Errorf("invalid block: block height of tx %d is not equal rx %d", blockHeight, blockPair.ResultsBlock.Header.BlockHeight())
	}
	if blockHeight > s.lastCommittedBlockHeight()+1 {
		return errors.Errorf("invalid block: future block height %d", blockHeight)
	}

	// block consensus
	var prevCommittedBlock *protocol.BlockPairContainer = nil
	if s.lastCommittedBlock != nil && blockHeight == s.lastCommittedBlockHeight()+1 {
		// in this case we also want to validate match to the prev (prev hashes)
		prevCommittedBlock = s.lastCommittedBlock
	}
	err := s.validateBlockConsensus(blockPair, prevCommittedBlock)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) nonLeaderCommitAndReplyUnsafe(blockPair *protocol.BlockPairContainer) error {
	// save the block to block storage
	err := s.saveToBlockStorage(blockPair)
	if err != nil {
		return err
	}

	// remember the block in our last committed state variable
	if blockPair.TransactionsBlock.Header.BlockHeight() == s.lastCommittedBlockHeight()+1 {
		s.lastCommittedBlock = blockPair
	}

	// sign the committed message we're about to send
	s.reporting.Infof("Replying committed with last committed height of %d", s.lastCommittedBlockHeight())
	status := (&gossipmessages.BenchmarkConsensusStatusBuilder{
		LastCommittedBlockHeight: s.lastCommittedBlockHeight(),
	}).Build()
	signedData := hash.CalcSha256(status.Raw())
	sig, err := signature.SignEd25519(s.config.NodePrivateKey(), signedData)
	if err != nil {
		return err
	}

	// send committed back to leader via gossip
	_, err = s.gossip.SendBenchmarkConsensusCommitted(&gossiptopics.BenchmarkConsensusCommittedInput{
		RecipientPublicKey: blockPair.ResultsBlock.BlockProof.BenchmarkConsensus().Sender().SenderPublicKey(),
		Message: &gossipmessages.BenchmarkConsensusCommittedMessage{
			Status: status,
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderPublicKey: s.config.NodePublicKey(),
				Signature:       sig,
			}).Build(),
		},
	})
	return err
}
