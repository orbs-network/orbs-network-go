// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"testing"
)

const RECENT_TIMESTAMP = primitives.TimestampNano(1505735343000000000)
const CURRENT_TIMESTAMP = primitives.TimestampNano(1505735309000000000)
const CURRENT_BLOCK_NUMBER = 938870
const NON_RECENT_TIMESTAMP = primitives.TimestampNano(1505734591000000000)
const NON_RECENT_BLOCK_NUMBER = 938774
const TOO_RECENT_TIMESTAMP = primitives.TimestampNano(1506109783000000000) // max + 1000 seconds
const TOO_RECENT_BLOCK_NUMBER = 999999

func TestEthereumGetBlockNumber(t *testing.T) {
	if !runningWithDocker() {
		t.Skip("Not running with Docker, Ganache is unavailable")
	}

	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newRpcEthereumConnectorHarness(parent.Logger, ConfigForExternalRPCConnection()).WithFakeTimeGetter()
			in := &services.EthereumGetBlockNumberInput{
				ReferenceTimestamp: RECENT_TIMESTAMP,
			}
			o, err := h.connector.EthereumGetBlockNumber(ctx, in)
			require.NoError(t, err, "failed getting block number from timestamp")
			require.EqualValues(t, CURRENT_BLOCK_NUMBER, o.EthereumBlockNumber, "block number on fake data mismatch")
		})
	})
}

func TestEthereumGetBlockTime(t *testing.T) {
	if !runningWithDocker() {
		t.Skip("Not running with Docker, Ganache is unavailable")
	}

	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newRpcEthereumConnectorHarness(parent.Logger, ConfigForExternalRPCConnection()).WithFakeTimeGetter()
			in := &services.EthereumGetBlockTimeInput{
				ReferenceTimestamp: RECENT_TIMESTAMP,
			}
			o, err := h.connector.EthereumGetBlockTime(ctx, in)
			require.NoError(t, err, "failed getting block number from timestamp")
			require.EqualValues(t, CURRENT_TIMESTAMP, o.EthereumTimestamp, "block time on fake data mismatch")
		})
	})
}

func TestEthereumGetBlockByTime(t *testing.T) {
	if !runningWithDocker() {
		t.Skip("Not running with Docker, Ganache is unavailable")
	}

	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newRpcEthereumConnectorHarness(parent.Logger, ConfigForExternalRPCConnection()).WithFakeTimeGetter()
			in := &services.EthereumGetBlockNumberByTimeInput{
				ReferenceTimestamp: RECENT_TIMESTAMP,
				EthereumTimestamp:  NON_RECENT_TIMESTAMP,
			}
			o, err := h.connector.EthereumGetBlockNumberByTime(ctx, in)
			require.NoError(t, err, "failed getting block number from timestamp")
			require.EqualValues(t, NON_RECENT_BLOCK_NUMBER, o.EthereumBlockNumber, "block time on fake data mismatch")
		})
	})
}

func TestEthereumGetBlockByTimeTooNewFails(t *testing.T) {
	if !runningWithDocker() {
		t.Skip("Not running with Docker, Ganache is unavailable")
	}

	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newRpcEthereumConnectorHarness(parent.Logger, ConfigForExternalRPCConnection()).WithFakeTimeGetter()
			in := &services.EthereumGetBlockNumberByTimeInput{
				ReferenceTimestamp: RECENT_TIMESTAMP,
				EthereumTimestamp:  TOO_RECENT_TIMESTAMP,
			}
			_, err := h.connector.EthereumGetBlockNumberByTime(ctx, in)
			require.Error(t, err, "should fail getting block number from a too recent timestamp")
		})
	})
}

func TestEthereumGetBlockTimeByNumber(t *testing.T) {
	if !runningWithDocker() {
		t.Skip("Not running with Docker, Ganache is unavailable")
	}

	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newRpcEthereumConnectorHarness(parent.Logger, ConfigForExternalRPCConnection()).WithFakeTimeGetter()
			in := &services.EthereumGetBlockTimeByNumberInput{
				ReferenceTimestamp:  RECENT_TIMESTAMP,
				EthereumBlockNumber: NON_RECENT_BLOCK_NUMBER,
			}
			o, err := h.connector.EthereumGetBlockTimeByNumber(ctx, in)
			require.NoError(t, err, "failed getting block number from timestamp")
			require.EqualValues(t, NON_RECENT_TIMESTAMP, o.EthereumTimestamp, "block time on fake data mismatch")
		})
	})
}

func TestEthereumGetBlockTimeByNumberTooNewFails(t *testing.T) {
	if !runningWithDocker() {
		t.Skip("Not running with Docker, Ganache is unavailable")
	}

	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newRpcEthereumConnectorHarness(parent.Logger, ConfigForExternalRPCConnection()).WithFakeTimeGetter()
			in := &services.EthereumGetBlockTimeByNumberInput{
				ReferenceTimestamp:  RECENT_TIMESTAMP,
				EthereumBlockNumber: TOO_RECENT_BLOCK_NUMBER,
			}
			_, err := h.connector.EthereumGetBlockTimeByNumber(ctx, in)
			require.Error(t, err, "should fail getting block number from a too recent timestamp")
		})
	})
}

func TestEthereumGetBlockAndTimeInterCalculations(t *testing.T) {
	if !runningWithDocker() {
		t.Skip("Not running with Docker, Ganache is unavailable")
	}

	test.WithContext(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newRpcEthereumConnectorHarness(parent.Logger, ConfigForExternalRPCConnection()).WithFakeTimeGetter()
			inCurrBlock := &services.EthereumGetBlockNumberInput{
				ReferenceTimestamp: RECENT_TIMESTAMP,
			}
			currBlock, err := h.connector.EthereumGetBlockNumber(ctx, inCurrBlock)
			require.NoError(t, err, "no err EthereumGetBlockNumber")
			inCurrTime := &services.EthereumGetBlockTimeInput{
				ReferenceTimestamp: RECENT_TIMESTAMP,
			}
			currTime, err := h.connector.EthereumGetBlockTime(ctx, inCurrTime)
			require.NoError(t, err, "no err EthereumGetBlockTime")

			inCalcTime := &services.EthereumGetBlockTimeByNumberInput{
				ReferenceTimestamp:  RECENT_TIMESTAMP,
				EthereumBlockNumber: currBlock.EthereumBlockNumber,
			}
			calcTime, err := h.connector.EthereumGetBlockTimeByNumber(ctx, inCalcTime)
			require.NoError(t, err, "no err EthereumGetBlockTimeByNumber")

			inCalcBlock := &services.EthereumGetBlockNumberByTimeInput{
				ReferenceTimestamp: RECENT_TIMESTAMP,
				EthereumTimestamp:  currTime.EthereumTimestamp,
			}
			calcBlock, err := h.connector.EthereumGetBlockNumberByTime(ctx, inCalcBlock)
			require.NoError(t, err, "no err EthereumGetBlockNumberByTimeInput")

			require.EqualValues(t, currBlock.EthereumBlockNumber, calcBlock.EthereumBlockNumber, "block numbers should match")
			require.EqualValues(t, currTime.EthereumTimestamp, calcTime.EthereumTimestamp, "block times should match")
		})
	})
}
