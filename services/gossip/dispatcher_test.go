package gossip

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestGossipMessageDispatcher_ReleasesPendingMessagesOnShutdown(t *testing.T) {
	test.WithContext(func(parent context.Context) {
		logger := log.DefaultTestingLogger(t)
		d := newMessageDispatcher(metric.NewRegistry())

		header := (&gossipmessages.HeaderBuilder{Topic: gossipmessages.HEADER_TOPIC_BLOCK_SYNC}).Build()
		ctx, cancel := context.WithCancel(parent)

		// this will bombard our topic with messages
		go func() {
			for {
				d.dispatch(ctx, logger, header, nil)
				time.Sleep(1 * time.Millisecond)
				if parent.Err() != nil {
					return
				}
			}
		}()

		d.runHandler(ctx, logger, header.Topic(), func(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
			// do nothing on purpose
		})

		cancel()

		topicChannel := d[header.Topic()]

		_, stillOpen := <-topicChannel.ch
		require.False(t, stillOpen, "channel was not closed")
		require.Empty(t, topicChannel.ch, "channel was not empty")
	})
}

func TestGossipMessageDispatcher_DoesNotPanicAfterChannelIsClosed(t *testing.T) {
	test.WithContext(func(parent context.Context) {
		logger := log.DefaultTestingLogger(t)
		d := newMessageDispatcher(metric.NewRegistry())

		header := (&gossipmessages.HeaderBuilder{Topic: gossipmessages.HEADER_TOPIC_BLOCK_SYNC}).Build()
		ctx, cancel := context.WithCancel(parent)
		cancel()

		d.runHandler(ctx, logger, header.Topic(), func(ctx context.Context, header *gossipmessages.Header, payloads [][]byte) {
			// do nothing on purpose
		})

		topicChannel := d[header.Topic()]

		<-topicChannel.ch // wait till channel is closed

		require.NotPanics(t, func() {
			d.dispatch(ctx, logger, header, nil)
		}, "dispatch panicked")
	})
}
