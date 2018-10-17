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
		sourcePK := keys.Ed25519KeyPairForTests(4).PublicKey()
		harness := newHarness(ctx).withNodeKey(sourcePK)
		harness.setupSomeBlocks(3)
		senderPK := keys.Ed25519KeyPairForTests(1).PublicKey()

		msg := builders.BlockAvailabilityRequestInput().
			WithSenderPublicKey(senderPK).
			WithFirstBlockHeight(1).
			WithLastCommittedBlockHeight(primitives.BlockHeight(2)).
			WithLastBlockHeight(primitives.BlockHeight(2)).
			Build()

		availabilityResponseVerifier := func(i interface{}) bool {
			response, ok := i.(*gossiptopics.BlockAvailabilityResponseInput)
			if !ok {
				require.Failf(t, "response type does not match", "", i)
			}

			require.Equal(t, senderPK, response.RecipientPublicKey, "public key does not match")
			require.Equal(t, sourcePK, response.Message.Sender.SenderPublicKey(), "source pk does not match")
			require.Equal(t, primitives.BlockHeight(1), response.Message.SignedBatchRange.FirstBlockHeight(), "first block height is not as expected")
			require.Equal(t, primitives.BlockHeight(3), response.Message.SignedBatchRange.LastCommittedBlockHeight(), "last committed block height is not as expected")
			require.Equal(t, primitives.BlockHeight(3), response.Message.SignedBatchRange.LastBlockHeight(), "last block height is not as expected")

			return true
		}

		harness.gossip.
			When("SendBlockAvailabilityResponse",
				mock.AnyIf("validating response of availability request", availabilityResponseVerifier)).
			Return(nil, nil).Times(1)

		_, err := harness.blockStorage.HandleBlockAvailabilityRequest(msg)

		require.NoError(t, err, "expecting a happy flow")

		harness.verifyMocks(t, 1)
	})
}

func TestSourceDoesNotRespondToAvailabilityRequestIfSourceIsBehindPetitioner(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		harness := newHarness(ctx)
		harness.expectCommitStateDiff()
		harness.commitBlock(builders.BlockPair().WithHeight(primitives.BlockHeight(1)).Build())

		harness.gossip.Never("SendBlockAvailabilityResponse", mock.Any)

		msg := builders.BlockAvailabilityRequestInput().WithLastCommittedBlockHeight(primitives.BlockHeight(20)).Build()
		_, err := harness.blockStorage.HandleBlockAvailabilityRequest(msg)

		require.NoError(t, err, "expecting a happy flow (without sending the response)")
		harness.verifyMocks(t, 1)
	})
}

func TestSourceIgnoresSendBlockAvailabilityRequestsIfFailedToRespond(t *testing.T) {

	test.WithContext(func(ctx context.Context) {
		harness := newHarness(ctx)
		harness.setupSomeBlocks(3)

		harness.gossip.When("SendBlockAvailabilityResponse", mock.Any).Return(nil, errors.New("gossip failure")).Times(1)
		msg := builders.BlockAvailabilityRequestInput().
			WithFirstBlockHeight(1).
			WithLastCommittedBlockHeight(primitives.BlockHeight(2)).
			WithLastBlockHeight(primitives.BlockHeight(2)).
			Build()
		_, err := harness.blockStorage.HandleBlockAvailabilityRequest(msg)

		require.Error(t, err, "expecting an error from the server event flow")

		harness.verifyMocks(t, 1)
	})
}

func TestSourceRespondsWithChunks(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		batchSize := uint32(10)
		harness := newHarness(ctx).withBatchSize(batchSize).withNodeKey(keys.Ed25519KeyPairForTests(4).PublicKey())
		lastBlock := 12
		harness.setupSomeBlocks(lastBlock)

		firstHeight := primitives.BlockHeight(1)
		lastHeight := primitives.BlockHeight(10) // hardcoding this, but it is a function of the batchSize

		msg := builders.BlockSyncRequestInput().
			WithSenderPublicKey(keys.Ed25519KeyPairForTests(1).PublicKey()).
			WithFirstBlockHeight(firstHeight).
			Build()

		chunksResponseVerifier := func(i interface{}) bool {
			response, ok := i.(*gossiptopics.BlockSyncResponseInput)
			if !ok {
				require.Failf(t, "response type does not match", "", i)
			}
			require.Len(t, response.Message.BlockPairs, int(batchSize), "actual batch size does not match config")
			require.Equal(t, keys.Ed25519KeyPairForTests(1).PublicKey(), response.RecipientPublicKey, "recipient pk is incorrect")
			require.Equal(t, firstHeight, response.Message.SignedChunkRange.FirstBlockHeight(), "first block height mismatch")
			require.Equal(t, lastHeight, response.Message.SignedChunkRange.LastBlockHeight(), "last block height mismatch")
			require.Equal(t, primitives.BlockHeight(lastBlock), response.Message.SignedChunkRange.LastCommittedBlockHeight(), "last committed block height mismatch")
			require.Equal(t, keys.Ed25519KeyPairForTests(4).PublicKey(), response.Message.Sender.SenderPublicKey(), "sender does not match config")
			require.Equal(t, msg.Message.SignedChunkRange.BlockType(), response.Message.SignedChunkRange.BlockType(), "block type does not match the request")

			return true
		}

		harness.gossip.When("SendBlockSyncResponse", mock.AnyIf("response should hold correct blocks", chunksResponseVerifier)).Return(nil, nil).Times(1)
		harness.blockStorage.HandleBlockSyncRequest(msg)
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

		harness := newHarness(ctx)
		harness.setupSomeBlocks(lastBlock)

		harness.gossip.Never("SendBlockSyncResponse", mock.Any)

		_, err := harness.blockStorage.HandleBlockSyncRequest(msg)

		require.Error(t, err, "expected source to return an error")
		harness.verifyMocks(t, 1)
	})
}
