package benchmarkconsensus

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/crypto/signature"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
)

func (s *service) nonLeaderHandleCommit(ctx context.Context, blockPair *protocol.BlockPairContainer) error {
	lastCommittedBlockHeight, lastCommittedBlock := s.getLastCommittedBlock()

	err := s.nonLeaderValidateBlock(blockPair, lastCommittedBlockHeight, lastCommittedBlock)
	if err != nil {
		return err
	}
	err = s.nonLeaderCommitAndReply(ctx, blockPair, lastCommittedBlockHeight, lastCommittedBlock)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) nonLeaderValidateBlock(blockPair *protocol.BlockPairContainer, lastCommittedBlockHeight primitives.BlockHeight, lastCommittedBlock *protocol.BlockPairContainer) error {
	// block height
	blockHeight := blockPair.TransactionsBlock.Header.BlockHeight()
	if blockHeight != blockPair.ResultsBlock.Header.BlockHeight() {
		return errors.Errorf("invalid block: block height of tx %s is not equal rx %s", blockHeight, blockPair.ResultsBlock.Header.BlockHeight())
	}
	if blockHeight > lastCommittedBlockHeight+1 {
		return errors.Errorf("invalid block: future block height %s", blockHeight)
	}

	// block consensus
	var prevCommittedBlockPair *protocol.BlockPairContainer = nil
	if lastCommittedBlock != nil && blockHeight == lastCommittedBlockHeight+1 {
		// in this case we also want to validate match to the prev (prev hashes)
		prevCommittedBlockPair = lastCommittedBlock
	}
	err := s.validateBlockConsensus(blockPair, prevCommittedBlockPair)
	if err != nil {
		return err
	}

	return nil
}

func (s *service) nonLeaderCommitAndReply(ctx context.Context, blockPair *protocol.BlockPairContainer, lastCommittedBlockHeight primitives.BlockHeight, lastCommittedBlock *protocol.BlockPairContainer) error {
	logger := s.logger.WithTags(trace.LogFieldFrom(ctx))

	// save the block to block storage
	err := s.saveToBlockStorage(ctx, blockPair)
	if err != nil {
		return err
	}

	// remember the block in our last committed state variable
	if blockPair.TransactionsBlock.Header.BlockHeight() == lastCommittedBlockHeight+1 {
		err = s.setLastCommittedBlock(blockPair, lastCommittedBlock)
		if err != nil {
			return err
		}
		// don't forget to update internal vars too since they may be used later on in the function
		lastCommittedBlock = blockPair
		lastCommittedBlockHeight = lastCommittedBlock.TransactionsBlock.Header.BlockHeight()
	}

	// sign the committed message we're about to send
	status := (&gossipmessages.BenchmarkConsensusStatusBuilder{
		LastCommittedBlockHeight: lastCommittedBlockHeight,
	}).Build()
	signedData := hash.CalcSha256(status.Raw())
	sig, err := signature.SignEd25519(s.config.NodePrivateKey(), signedData)
	if err != nil {
		return err
	}

	// prepare the message
	message := &gossipmessages.BenchmarkConsensusCommittedMessage{
		Status: status,
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderPublicKey: s.config.NodePublicKey(),
			Signature:       sig,
		}).Build(),
	}

	// send committed back to leader via gossip
	recipient := blockPair.ResultsBlock.BlockProof.BenchmarkConsensus().Sender().SenderPublicKey()
	logger.Info("replying committed with last committed height", log.BlockHeight(lastCommittedBlockHeight), log.Stringable("signed-data", signedData))
	_, err = s.gossip.SendBenchmarkConsensusCommitted(ctx, &gossiptopics.BenchmarkConsensusCommittedInput{
		RecipientPublicKey: recipient,
		Message:            message,
	})
	return err
}
