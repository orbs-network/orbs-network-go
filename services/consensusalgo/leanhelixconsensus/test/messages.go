// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"github.com/orbs-network/crypto-lib-go/crypto/digest"
	"github.com/orbs-network/lean-helix-go/services/interfaces"
	"github.com/orbs-network/lean-helix-go/services/messagesfactory"
	lhprimitives "github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/orbs-network-go/services/consensusalgo/leanhelixconsensus"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
)

func APreprepareMessage(
	instanceId lhprimitives.InstanceId,
	keyManager interfaces.KeyManager,
	senderMemberId lhprimitives.MemberId,
	blockHeight lhprimitives.BlockHeight,
	view lhprimitives.View,
	block interfaces.Block) *interfaces.PreprepareMessage {

	pair := leanhelixconsensus.FromLeanHelixBlock(block)
	messageFactory := messagesfactory.NewMessageFactory(instanceId, keyManager, senderMemberId, 0)
	return messageFactory.CreatePreprepareMessage(blockHeight, view, block, []byte(digest.CalcBlockHash(pair.TransactionsBlock, pair.ResultsBlock)))
}

func ACommitMessage(
	instanceId lhprimitives.InstanceId,
	keyManager interfaces.KeyManager,
	senderMemberId lhprimitives.MemberId,
	blockHeight lhprimitives.BlockHeight,
	view lhprimitives.View,
	block interfaces.Block,
	randomSeed uint64) *interfaces.CommitMessage {

	pair := leanhelixconsensus.FromLeanHelixBlock(block)
	messageFactory := messagesfactory.NewMessageFactory(instanceId, keyManager, senderMemberId, randomSeed)
	return messageFactory.CreateCommitMessage(blockHeight, view, []byte(digest.CalcBlockHash(pair.TransactionsBlock, pair.ResultsBlock)))
}

func generatePreprepareMessage(instanceId lhprimitives.InstanceId, block interfaces.Block, blockHeight uint64, view lhprimitives.View, senderNodeAddress primitives.NodeAddress, keyManager interfaces.KeyManager) *interfaces.ConsensusRawMessage {
	senderMemberId := lhprimitives.MemberId(senderNodeAddress)
	return APreprepareMessage(instanceId, keyManager, senderMemberId, lhprimitives.BlockHeight(blockHeight), view, block).ToConsensusRawMessage()
}

func generateCommitMessage(instanceId lhprimitives.InstanceId, block interfaces.Block, blockHeight uint64, view lhprimitives.View, senderNodeAddress primitives.NodeAddress, randomSeed uint64, keyManager interfaces.KeyManager) *interfaces.ConsensusRawMessage {
	senderMemberId := lhprimitives.MemberId(senderNodeAddress)
	return ACommitMessage(instanceId, keyManager, senderMemberId, lhprimitives.BlockHeight(blockHeight), view, block, randomSeed).ToConsensusRawMessage()
}
