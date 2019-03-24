// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"github.com/orbs-network/lean-helix-go/services/interfaces"
	"github.com/orbs-network/lean-helix-go/spec/types/go/primitives"
	"github.com/orbs-network/lean-helix-go/test/builders"
	"github.com/orbs-network/lean-helix-go/test/mocks"
)

func generatePreprepareMessage(instanceId primitives.InstanceId, blockHeight uint64, view primitives.View, senderMemberIdStr string) *interfaces.ConsensusRawMessage {
	senderMemberId := primitives.MemberId(senderMemberIdStr)
	keyManager := mocks.NewMockKeyManager(senderMemberId)
	block := mocks.ABlock(interfaces.GenesisBlock)
	return builders.APreprepareMessage(instanceId, keyManager, senderMemberId, primitives.BlockHeight(blockHeight), view, block).ToConsensusRawMessage()
}
