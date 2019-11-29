// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package tcp

import (
	"context"
	"fmt"
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/logfields"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/instrumentation/trace"
	"github.com/orbs-network/orbs-network-go/services/gossip/adapter"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"net"
	"time"
)

type timingsConfig interface {
	GossipNetworkTimeout() time.Duration
	GossipReconnectInterval() time.Duration
	GossipConnectionKeepAliveInterval() time.Duration
}

type outgoingConnection struct {
	logger          log.Logger
	metricRegistry  metric.Registry
	config          timingsConfig
	sharedMetrics   *outgoingConnectionMetrics // TODO this is smelly, see how we can restructure metrics so that an outgoing connection doesn't have to share the parent metrics
	roundtripMetric *metric.Histogram
	queue           *transportQueue
	peerHexAddress  string
	cancel          context.CancelFunc

	sendErrors      *metric.Gauge
	sendQueueErrors *metric.Gauge

	closed chan struct{}
}

func newOutgoingConnection(peer config.GossipPeer, parentLogger log.Logger, metricFactory metric.Registry, sharedMetrics *outgoingConnectionMetrics, transportConfig timingsConfig) *outgoingConnection {
	networkAddress := fmt.Sprintf("%s:%d", peer.GossipEndpoint(), peer.GossipPort())
	hexAddressSliceForLogging := peer.HexOrbsAddress()[:6]

	logger := parentLogger.WithTags(log.String("peer-node-address", hexAddressSliceForLogging), log.String("peer-network-address", networkAddress))

	queue := NewTransportQueue(SEND_QUEUE_MAX_BYTES, SEND_QUEUE_MAX_MESSAGES, metricFactory, hexAddressSliceForLogging)
	queue.networkAddress = networkAddress
	queue.Disable() // until connection is established

	client := &outgoingConnection{
		logger:          logger,
		sharedMetrics:   sharedMetrics,
		metricRegistry:  metricFactory,
		roundtripMetric: metricFactory.NewLatency(fmt.Sprintf("Gossip.OutgoingConnection.Roundtrip.%s", hexAddressSliceForLogging), time.Hour),
		config:          transportConfig,
		queue:           queue,
		peerHexAddress:  hexAddressSliceForLogging,
		sendErrors:      metricFactory.NewGauge(fmt.Sprintf("Gossip.OutgoingConnection.SendError.%s.Count", hexAddressSliceForLogging)),
		sendQueueErrors: metricFactory.NewGauge(fmt.Sprintf("Gossip.OutgoingConnection.EnqueueErrors.%s.Count", hexAddressSliceForLogging)),
	}

	return client
}

func (c *outgoingConnection) connect(parent context.Context) {
	ctx, cancel := context.WithCancel(parent)
	c.cancel = cancel

	handle := govnr.Forever(ctx, fmt.Sprintf("TCP client for %s", c.peerHexAddress), logfields.GovnrErrorer(c.logger), func() {
		c.connectionMainLoop(ctx)
	})
	c.closed = handle.Done()
	handle.MarkSupervised() //TODO use real supervision?
}

func (c *outgoingConnection) disconnect() chan struct{} {
	c.cancel()
	return c.closed
}

func (c *outgoingConnection) connectionMainLoop(parentCtx context.Context) {
	for {
		if parentCtx.Err() != nil {
			return // because otherwise the continue statement below could prevent us from ever shutting down
		}
		ctx := trace.NewContext(parentCtx, fmt.Sprintf("Gossip.Transport.TCP.Client.%s", c.peerHexAddress))
		logger := c.logger.WithTags(trace.LogFieldFrom(ctx))

		logger.Info("attempting outgoing transport connection")
		conn, err := net.DialTimeout("tcp", c.queue.networkAddress, c.config.GossipNetworkTimeout())

		if err != nil {
			logger.Info("cannot connect to gossip peer endpoint")
			time.Sleep(c.config.GossipReconnectInterval())
			continue
		}

		if !c.handleOutgoingConnection(ctx, conn) {
			return
		}
	}
}

func isTimeoutError(err error) bool {
	e, ok := err.(net.Error)
	return ok && e.Timeout()
}

func tryRead(errCh chan error) (error, bool) {
	select {
	case err := <-errCh:
		return err, true
	default:
		return nil, false
	}
}

// returns true if should attempt reconnect on error
func (c *outgoingConnection) handleOutgoingConnection(ctx context.Context, conn net.Conn) bool {
	logger := c.logger.WithTags(trace.LogFieldFrom(ctx), log.Stringable("local-address", conn.LocalAddr()))
	logger.Info("successful outgoing gossip transport connection")

	c.sharedMetrics.activeCount.Inc()
	defer c.sharedMetrics.activeCount.Dec()

	c.queue.OnNewConnection(ctx)
	defer c.queue.Disable()

	defer conn.Close() // we only exit this function when this connection has errored, or if we're disconnecting, so we can safely defer close()

	transmissionTimeQueue := newTransmissionTimeQueue()

	ackThreadErrChan := make(chan error)
	childCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	c.receiveAcks(childCtx, conn, transmissionTimeQueue, func(err error) { ackThreadErrChan <- err })

	for {
		if err, ok := tryRead(ackThreadErrChan); ok && err != nil { // if receiveAcks closed without an error, then we are closing as well
			logger.Info("connection closing due to error on ack thread")
			return c.reconnectAfterSocketError(logger, err)
		}

		if data := c.popMessageFromQueue(ctx); data != nil {
			// got data from queue
			transmissionTimeQueue.push(time.Now()) // must push before sending to avoid race with ack thread
			err := c.sendToSocket(ctx, conn, data)
			if err != nil {
				logger.Info("connection closing due to socket error")
				return c.reconnectAfterSocketError(logger, err)
			} // else - continue looping

		} else if shouldKeepAlive(ctx) {
			err := c.sendKeepAlive(ctx, conn)
			if err != nil {
				logger.Info("connection closing due to keep alive error")
				return c.reconnectAfterKeepAliveFailure(logger, err)
			}

		} else {
			// parent ctx is closed, we're disconnecting
			logger.Info("connection closing due to requested disconnect")
			return c.onDisconnect(logger)
		}
	}
}

func (c *outgoingConnection) popMessageFromQueue(ctx context.Context) *adapter.TransportData {
	ctxWithKeepAliveTimeout, cancelCtxWithKeepAliveTimeout := context.WithTimeout(ctx, c.config.GossipConnectionKeepAliveInterval())
	defer cancelCtxWithKeepAliveTimeout()

	return c.queue.Pop(ctxWithKeepAliveTimeout)
}

// if this context is not closed, we send keep alive; otherwise - we're asked to disconnect
func shouldKeepAlive(ctx context.Context) bool {
	return ctx.Err() == nil
}

func (c *outgoingConnection) onDisconnect(logger log.Logger) bool {
	logger.Info("client loop stopped since a disconnect was requested (topology change or system shutdown)")
	c.metricRegistry.Remove(c.sendErrors)
	c.metricRegistry.Remove(c.sendQueueErrors)
	c.metricRegistry.Remove(c.queue.usagePercentageMetric)
	return false
}

func (c *outgoingConnection) reconnectAfterKeepAliveFailure(logger log.Logger, err error) bool {
	c.sharedMetrics.KeepaliveErrors.Inc()
	logger.Info("failed sending keepalive, reconnecting", log.Error(err))
	return true
}

func (c *outgoingConnection) reconnectAfterSocketError(logger log.Logger, err error) bool {
	c.sharedMetrics.sendErrors.Inc() //TODO remove, replaced by following metric
	c.sendErrors.Inc()
	logger.Info("failed sending transport data, reconnecting", log.Error(err))
	return true
}

func (c *outgoingConnection) addDataToOutgoingPeerQueue(ctx context.Context, data *adapter.TransportData) {
	err := c.queue.Push(data)
	if err != nil {
		c.sharedMetrics.sendQueueErrors.Inc() //TODO remove, replaced by following metric
		c.sendQueueErrors.Inc()
		c.logger.Info("direct transport send queue error", log.Error(err), trace.LogFieldFrom(ctx))
	}
}

func (c *outgoingConnection) sendToSocket(ctx context.Context, conn net.Conn, data *adapter.TransportData) error {
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

func (c *outgoingConnection) sendKeepAlive(ctx context.Context, conn net.Conn) error {
	timeout := c.config.GossipNetworkTimeout()
	zeroBuffer := make([]byte, 4)

	// send zero num payloads
	err := write(ctx, conn, zeroBuffer, timeout)
	if err != nil {
		return err
	}

	return nil
}

func (c *outgoingConnection) receiveAcks(ctx context.Context, conn net.Conn, transmissionTimeRecorder *TransmissionTimeQueue, onClose func(err error)) {
	govnr.Once(logfields.GovnrErrorer(c.logger), func() {
		var err error
		defer func() {
			panicErr := recover()
			if panicErr != nil {
				err = fmt.Errorf("reciveAcks thread panicked: %s", panicErr)
			}
			if ctx.Err() == nil {
				c.logger.Info("shutting down ack thread:", log.Error(err))
				onClose(err)
			} else {
				onClose(nil)
			}
		}()
		for {
			_, err = readTotal(ctx, conn, uint32(len(ACK_BUFFER)), c.config.GossipConnectionKeepAliveInterval())
			if err != nil {
				if !isTimeoutError(err) {
					return // error is passed to onClose by defer
				}
			} else {
				t, ok := transmissionTimeRecorder.pop()
				if ok {
					c.roundtripMetric.RecordSince(t)
				}
			}
		}
	})
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
