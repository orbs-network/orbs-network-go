// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"fmt"
	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"net"
	"time"
)

type clientConnectionConfig interface {
	GossipNetworkTimeout() time.Duration
	GossipReconnectInterval() time.Duration
	GossipConnectionKeepAliveInterval() time.Duration
}

type clientConnection struct {
	logger        log.Logger
	config        clientConnectionConfig
	sharedMetrics *metrics
	queue         *transportQueue

	sendErrors      *metric.Gauge
	sendQueueErrors *metric.Gauge
}

func newClientConnection(peerNodeAddress string, peer config.GossipPeer, logger log.Logger, metricFactory metric.Registry, sharedMetrics *metrics, transportConfig config.GossipTransportConfig) *clientConnection {
	peerAddress := fmt.Sprintf("%s:%d", peer.GossipEndpoint(), peer.GossipPort())
	queue := NewTransportQueue(SEND_QUEUE_MAX_BYTES, SEND_QUEUE_MAX_MESSAGES, metricFactory, peerNodeAddress)
	queue.networkAddress = peerAddress
	queue.Disable() // until connection is established

	client := &clientConnection{
		logger:          logger,
		sharedMetrics:   sharedMetrics,
		config:          transportConfig,
		queue:           queue,
		sendErrors:      metricFactory.NewGauge(fmt.Sprintf("Gossip.OutgoingConnection.SendError.%s.Count", peerNodeAddress)),
		sendQueueErrors: metricFactory.NewGauge(fmt.Sprintf("Gossip.OutgoingConnection.EnqueueErrors.%s.Count", peerNodeAddress)),
	}

	return client
}

func (c *clientConnection) connect(ctx context.Context) {

	supervised.GoForever(ctx, c.logger, func() {
		c.clientMainLoop(ctx, c.queue) // avoid referencing queue map not under lock
	})
}

func (c *clientConnection) clientMainLoop(parentCtx context.Context, queue *transportQueue) {
	for {
		ctx := trace.NewContext(parentCtx, fmt.Sprintf("Gossip.Transport.TCP.Client.%s", queue.networkAddress))
		c.logger.Info("attempting outgoing transport connection", log.String("peer", queue.networkAddress), trace.LogFieldFrom(ctx))
		conn, err := net.DialTimeout("tcp", queue.networkAddress, c.config.GossipNetworkTimeout())

		if err != nil {
			c.logger.Info("cannot connect to gossip peer endpoint", log.String("peer", queue.networkAddress), trace.LogFieldFrom(ctx))
			time.Sleep(c.config.GossipReconnectInterval())
			continue
		}

		if !c.clientHandleOutgoingConnection(ctx, conn, queue) {
			return
		}
	}
}

// returns true if should attempt reconnect on error
func (c *clientConnection) clientHandleOutgoingConnection(ctx context.Context, conn net.Conn, queue *transportQueue) bool {
	c.logger.Info("successful outgoing gossip transport connection", log.String("peer", queue.networkAddress), trace.LogFieldFrom(ctx))
	c.sharedMetrics.activeOutgoingConnections.Inc()
	defer c.sharedMetrics.activeOutgoingConnections.Dec()
	queue.Clear(ctx)
	queue.Enable()
	defer queue.Disable()

	for {

		ctxWithKeepAliveTimeout, cancelCtxWithKeepAliveTimeout := context.WithTimeout(ctx, c.config.GossipConnectionKeepAliveInterval())
		data := queue.Pop(ctxWithKeepAliveTimeout)
		cancelCtxWithKeepAliveTimeout()

		if data != nil {

			// ctxWithKeepAliveTimeout not closed, so no keep alive timeout nor system shutdown
			// meaning do a regular send (we have data)
			err := c.sendTransportData(ctx, conn, data)
			if err != nil {
				c.sharedMetrics.outgoingConnectionSendErrors.Inc() //TODO remove, replaced by following metric
				c.sendErrors.Inc()
				c.logger.Info("failed sending transport data, reconnecting", log.Error(err), log.String("peer", queue.networkAddress), trace.LogFieldFrom(ctx))
				conn.Close()
				return true
			}

		} else {
			// ctxWithKeepAliveTimeout is closed, so either keep alive timeout or system shutdown
			if ctx.Err() == nil {

				// parent ctx not closed, so no system shutdown
				// meaning keep alive timeout
				err := c.sendKeepAlive(ctx, conn)
				if err != nil {
					c.sharedMetrics.outgoingConnectionKeepaliveErrors.Inc()
					c.logger.Info("failed sending keepalive, reconnecting", log.Error(err), log.String("peer", queue.networkAddress), trace.LogFieldFrom(ctx))
					conn.Close()
					return true
				}

			} else {

				// parent ctx is closed, so system shutdown
				// meaning cleanup and exit
				c.logger.Info("client loop stopped since server is shutting down", trace.LogFieldFrom(ctx))
				conn.Close()
				return false

			}
		}
	}
}

func (c *clientConnection) addDataToOutgoingPeerQueue(data *adapter.TransportData) {
	err := c.queue.Push(data)
	if err != nil {
		c.sharedMetrics.outgoingConnectionSendQueueErrors.Inc() //TODO remove, replaced by following metric
		c.sendQueueErrors.Inc()
		c.logger.Info("direct transport send queue error", log.Error(err), log.String("peer", c.queue.networkAddress))
	}
	c.sharedMetrics.outgoingMessageSize.Record(int64(data.TotalSize()))
}

func (c *clientConnection) sendTransportData(ctx context.Context, conn net.Conn, data *adapter.TransportData) error {
	timeout := c.config.GossipNetworkTimeout()
	zeroBuffer := make([]byte, 4)
	sizeBuffer := make([]byte, 4)

	// send num payloads
	membuffers.WriteUint32(sizeBuffer, uint32(len(data.Payloads)))
	err := write(ctx, conn, sizeBuffer, timeout)
	if err != nil {
		return err
	}

	for _, payload := range data.Payloads {
		// send payload size
		membuffers.WriteUint32(sizeBuffer, uint32(len(payload)))
		err := write(ctx, conn, sizeBuffer, timeout)
		if err != nil {
			return err
		}

		// send payload data
		err = write(ctx, conn, payload, timeout)
		if err != nil {
			return err
		}

		// send padding
		paddingSize := calcPaddingSize(uint32(len(payload)))
		if paddingSize > 0 {
			err = write(ctx, conn, zeroBuffer[:paddingSize], timeout)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (c *clientConnection) sendKeepAlive(ctx context.Context, conn net.Conn) error {
	timeout := c.config.GossipNetworkTimeout()
	zeroBuffer := make([]byte, 4)

	// send zero num payloads
	err := write(ctx, conn, zeroBuffer, timeout)
	if err != nil {
		return err
	}

	return nil
}

func write(ctx context.Context, conn net.Conn, buffer []byte, timeout time.Duration) error {
	// TODO(https://github.com/orbs-network/orbs-network-go/issues/182): consider whether the right approach is to poll context this way or have a single watchdog goroutine that closes all active connections when context is cancelled
	// make sure context is still open
	err := ctx.Err()
	if err != nil {
		return err
	}

	err = conn.SetWriteDeadline(time.Now().Add(timeout))
	if err != nil {
		return err
	}
	written, err := conn.Write(buffer)
	if written != len(buffer) {
		if err == nil {
			return errors.Errorf("attempted to write %d bytes but only wrote %d", len(buffer), written)
		} else {
			return errors.Wrapf(err, "attempted to write %d bytes but only wrote %d", len(buffer), written)
		}
	}
	return nil
}
