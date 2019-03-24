// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"errors"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestSourceRespondToAvailabilityRequests(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		sourceAddress := keys.EcdsaSecp256K1KeyPairForTests(4).NodeAddress()
		harness := newBlockStorageHarness(t).withNodeAddress(sourceAddress).withSyncBroadcast(1).start(ctx)
		harness.commitSomeBlocks(ctx, 3)
		senderAddress := keys.EcdsaSecp256K1KeyPairForTests(1).NodeAddress()

		msg := builders.BlockAvailabilityRequestInput().
			WithSenderNodeAddress(senderAddress).
			WithFirstBlockHeight(1).
			WithLastCommittedBlockHeight(primitives.BlockHeight(2)).
			WithLastBlockHeight(primitives.BlockHeight(2)).
			Build()

		availabilityResponseVerifier := func(i interface{}) bool {
			response, ok := i.(*gossiptopics.BlockAvailabilityResponseInput)
			if !ok {
				require.Failf(t, "response type does not match", "", i)
			}

			require.Equal(t, senderAddress, response.RecipientNodeAddress, "public key does not match")
			require.Equal(t, sourceAddress, response.Message.Sender.SenderNodeAddress(), "source nodeAddress does not match")
			require.Equal(t, primitives.BlockHeight(1), response.Message.SignedBatchRange.FirstBlockHeight(), "first block height is not as expected")
			require.Equal(t, primitives.BlockHeight(3), response.Message.SignedBatchRange.LastCommittedBlockHeight(), "last committed block height is not as expected")
			require.Equal(t, primitives.BlockHeight(3), response.Message.SignedBatchRange.LastBlockHeight(), "last block height is not as expected")

			return true
		}

		harness.gossip.
			When("SendBlockAvailabilityResponse", mock.Any, mock.AnyIf("validating response of availability request", availabilityResponseVerifier)).
			Return(nil, nil).Times(1)

		_, err := harness.blockStorage.HandleBlockAvailabilityRequest(ctx, msg)

		require.NoError(t, err, "expecting a happy flow")

		harness.verifyMocks(t, 1)
	})
}

func TestSourceDoesNotRespondToAvailabilityRequestIfSourceIsBehindPetitioner(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).withSyncBroadcast(1).start(ctx)
		harness.commitBlock(ctx, builders.BlockPair().WithHeight(primitives.BlockHeight(1)).Build())

		harness.gossip.Never("SendBlockAvailabilityResponse", mock.Any, mock.Any)

		msg := builders.BlockAvailabilityRequestInput().WithLastCommittedBlockHeight(primitives.BlockHeight(20)).Build()
		_, err := harness.blockStorage.HandleBlockAvailabilityRequest(ctx, msg)

		require.NoError(t, err, "expecting a happy flow (without sending the response)")
		harness.verifyMocks(t, 1)
	})
}

func TestSourceIgnoresSendBlockAvailabilityRequestsIfFailedToRespond(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newBlockStorageHarness(t).withSyncBroadcast(1).start(ctx)
		harness.commitSomeBlocks(ctx, 3)

		harness.gossip.When("SendBlockAvailabilityResponse", mock.Any, mock.Any).Return(nil, errors.New("gossip failure")).Times(1)
		msg := builders.BlockAvailabilityRequestInput().
			WithFirstBlockHeight(1).
			WithLastCommittedBlockHeight(primitives.BlockHeight(2)).
			WithLastBlockHeight(primitives.BlockHeight(2)).
			Build()
		_, err := harness.blockStorage.HandleBlockAvailabilityRequest(ctx, msg)

		require.Error(t, err, "expecting an error from the server event flow")

		harness.verifyMocks(t, 1)
	})
}

func TestSourceRespondsWithChunks(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		batchSize := uint32(10)
		harness := newBlockStorageHarness(t).
			withBatchSize(batchSize).
			withNodeAddress(keys.EcdsaSecp256K1KeyPairForTests(4).NodeAddress()).
			withSyncBroadcast(1).
			start(ctx)

		lastBlock := 12
		harness.commitSomeBlocks(ctx, lastBlock)

		firstHeight := primitives.BlockHeight(1)
		lastHeight := primitives.BlockHeight(10) // hardcoding this, but it is a function of the batchSize

		msg := builders.BlockSyncRequestInput().
			WithSenderNodeAddress(keys.EcdsaSecp256K1KeyPairForTests(1).NodeAddress()).
			WithFirstBlockHeight(firstHeight).
			Build()

		chunksResponseVerifier := func(i interface{}) bool {
			response, ok := i.(*gossiptopics.BlockSyncResponseInput)
			if !ok {
				require.Failf(t, "response type does not match", "", i)
			}
			require.Len(t, response.Message.BlockPairs, int(batchSize), "actual batch size does not match config")
			require.Equal(t, keys.EcdsaSecp256K1KeyPairForTests(1).NodeAddress(), response.RecipientNodeAddress, "recipient nodeAddress is incorrect")
			require.Equal(t, firstHeight, response.Message.SignedChunkRange.FirstBlockHeight(), "first block height mismatch")
			require.Equal(t, lastHeight, response.Message.SignedChunkRange.LastBlockHeight(), "last block height mismatch")
			require.Equal(t, primitives.BlockHeight(lastBlock), response.Message.SignedChunkRange.LastCommittedBlockHeight(), "last committed block height mismatch")
			require.Equal(t, keys.EcdsaSecp256K1KeyPairForTests(4).NodeAddress(), response.Message.Sender.SenderNodeAddress(), "sender does not match config")
			require.Equal(t, msg.Message.SignedChunkRange.BlockType(), response.Message.SignedChunkRange.BlockType(), "block type does not match the request")

			return true
		}

		harness.gossip.When("SendBlockSyncResponse", mock.Any, mock.AnyIf("response should hold correct blocks", chunksResponseVerifier)).Return(nil, nil).Times(1)
		harness.blockStorage.HandleBlockSyncRequest(ctx, msg)
		harness.verifyMocks(t, 1)
	})
}

func TestSourceIgnoresBlockSyncRequestIfSourceIsBehind(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		lastBlock := 10
		firstHeight := primitives.BlockHeight(lastBlock + 1)
		lastHeight := primitives.BlockHeight(lastBlock)

		msg := builders.BlockSyncRequestInput().
			WithFirstBlockHeight(firstHeight).
			WithLastCommittedBlockHeight(lastHeight).
			Build()

		harness := newBlockStorageHarness(t).withSyncBroadcast(1).start(ctx)
		harness.commitSomeBlocks(ctx, lastBlock)

		harness.gossip.Never("SendBlockSyncResponse", mock.Any, mock.Any)

		_, err := harness.blockStorage.HandleBlockSyncRequest(ctx, msg)

		require.Error(t, err, "expected source to return an error")
		harness.verifyMocks(t, 1)
	})
}
