// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package benchmarkconsensus

import (
	"context"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/pkg/errors"
	"time"
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
	s.metrics.lastCommittedTime.Update(time.Now().UnixNano())

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
	sig, err := digest.SignAsNode(s.config.NodePrivateKey(), status.Raw())
	if err != nil {
		return err
	}

	// prepare the message
	message := &gossipmessages.BenchmarkConsensusCommittedMessage{
		Status: status,
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: s.config.NodeAddress(),
			Signature:         sig,
		}).Build(),
	}

	// send committed back to leader via gossip
	signerIterator := blockPair.ResultsBlock.BlockProof.BenchmarkConsensus().NodesIterator()
	if !signerIterator.HasNext() {
		return errors.New("proof does not have a signer, unclear who to reply to")
	}
	recipient := signerIterator.NextNodes().SenderNodeAddress()
	logger.Info("replying committed with last committed height", log.BlockHeight(lastCommittedBlockHeight), log.Bytes("signed-data", status.Raw()))
	_, err = s.gossip.SendBenchmarkConsensusCommitted(ctx, &gossiptopics.BenchmarkConsensusCommittedInput{
		RecipientNodeAddress: recipient,
		Message:              message,
	})
	return err
}
