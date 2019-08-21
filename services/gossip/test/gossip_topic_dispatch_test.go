package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/gossip"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter/memory"
	"github.com/orbs-network/orbs-network-go/services/gossip/codec"
	"github.com/orbs-network/orbs-network-go/services/transactionpool"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

type conf struct {
}

func (c *conf) NodeAddress() primitives.NodeAddress {
	return []byte{0x01}
}

func (c *conf) VirtualChainId() primitives.VirtualChainId {
	return 42
}

func TestDifferentTopicsDoNotBlockEachOtherForSamePeer(t *testing.T) {
	test.WithConcurrencyHarness(t, func(ctx context.Context, harness *test.ConcurrencyHarness) {
		nodeAddresses := []primitives.NodeAddress{{0x01}, {0x02}}
		cfg := &conf{}

		genesisValidatorNodes := make(map[string]config.ValidatorNode)
		for _, address := range nodeAddresses {
			genesisValidatorNodes[address.KeyForMap()] = config.NewHardCodedValidatorNode(primitives.NodeAddress(address))
		}
		transport := memory.NewTransport(ctx, harness.Logger, genesisValidatorNodes)
		g := gossip.NewGossip(ctx, transport, cfg, harness.Logger, metric.NewRegistry())

		harness.Supervise(transport)
		harness.Supervise(g)

		trh := &gossiptopics.MockTransactionRelayHandler{}
		bsh := &gossiptopics.MockBlockSyncHandler{}

		g.RegisterTransactionRelayHandler(trh)
		g.RegisterBlockSyncHandler(bsh)

		blockSyncNotify := make(chan struct{})
		bsh.When("HandleBlockAvailabilityRequest", mock.Any, mock.Any).Call(func(nested context.Context, input *gossiptopics.BlockAvailabilityRequestInput) {
			close(blockSyncNotify)
			time.Sleep(1 * time.Hour)
		})

		trh.When("HandleForwardedTransactions", mock.Any, mock.Any).Times(1).Return(&gossiptopics.EmptyOutput{}, nil)

		require.NoError(t, transport.Send(ctx, &adapter.TransportData{
			SenderNodeAddress:      []byte{0x02},
			RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
			RecipientNodeAddresses: []primitives.NodeAddress{cfg.NodeAddress()},
			Payloads:               aBlockSyncRequest(t),
		}))

		<-blockSyncNotify

		require.NoError(t, transport.Send(ctx, &adapter.TransportData{
			SenderNodeAddress:      []byte{0x02},
			RecipientMode:          gossipmessages.RECIPIENT_LIST_MODE_LIST,
			RecipientNodeAddresses: []primitives.NodeAddress{cfg.NodeAddress()},
			Payloads:               aTransactionRelayRequest(t),
		}))

		require.NoError(t, test.EventuallyVerify(1*time.Second, trh, bsh), "mocks were not invoked as expected")

	})
}

func aBlockSyncRequest(t testing.TB) [][]byte {
	header := &gossipmessages.HeaderBuilder{
		Topic:          gossipmessages.HEADER_TOPIC_BLOCK_SYNC,
		BlockSync:      gossipmessages.BLOCK_SYNC_AVAILABILITY_REQUEST,
		RecipientMode:  gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		VirtualChainId: 42,
	}
	payloads, err := codec.EncodeBlockAvailabilityRequest(header.Build(), &gossipmessages.BlockAvailabilityRequestMessage{
		SignedBatchRange: (&gossipmessages.BlockSyncRangeBuilder{
			BlockType:                gossipmessages.BLOCK_TYPE_BLOCK_PAIR,
			FirstBlockHeight:         1001,
			LastBlockHeight:          2001,
			LastCommittedBlockHeight: 3001,
		}).Build(),
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: []byte{0x01, 0x02, 0x03},
			Signature:         []byte{0x04, 0x05, 0x06},
		}).Build(),
	})

	require.NoError(t, err, "encoding failed")
	return payloads
}

func aTransactionRelayRequest(t testing.TB) [][]byte {
	header := (&gossipmessages.HeaderBuilder{
		Topic:            gossipmessages.HEADER_TOPIC_TRANSACTION_RELAY,
		TransactionRelay: gossipmessages.TRANSACTION_RELAY_FORWARDED_TRANSACTIONS,
		RecipientMode:    gossipmessages.RECIPIENT_LIST_MODE_BROADCAST,
		VirtualChainId:   42,
	}).Build()

	payloads, err := codec.EncodeForwardedTransactions(header, &gossipmessages.ForwardedTransactionsMessage{
		Sender: (&gossipmessages.SenderSignatureBuilder{
			SenderNodeAddress: []byte{0x01, 0x02, 0x03},
			Signature:         []byte{0x04, 0x05, 0x06},
		}).Build(),
		SignedTransactions: transactionpool.Transactions{builders.TransferTransaction().Build()},
	})
	require.NoError(t, err, "encoding failed")
	return payloads
}
