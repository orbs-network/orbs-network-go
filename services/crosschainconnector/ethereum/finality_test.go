// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package ethereum

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/timestampfinder"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/scribe/log"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const RECENT_TIMESTAMP = primitives.TimestampNano(1505735343000000000)
const FINALITY_BLOCK_TIME = primitives.TimestampNano(1505734591000000000)
const RECENT_BLOCK_NUMBER = 938874
const FINALITY_BLOCK_NUMBER = 938774
const FINALITY_BLOCKS = 100

type harness struct {
	service *service
}

func newHarness(logger log.Logger, fct time.Duration, fbc uint32) *harness {
	cfg := &finalityConfig{fct, fbc}

	btg := timestampfinder.NewFakeBlockTimeGetter(logger)
	finder := timestampfinder.NewTimestampFinder(btg, logger, metric.NewRegistry())

	blockTimeGetter := timestampfinder.NewFakeBlockTimeGetter(logger)
	s := &service{
		connection:      nil,
		blockTimeGetter: blockTimeGetter,
		timestampFinder: finder,
		logger:          logger,
		config:          cfg,
	}

	h := &harness{
		service: s,
	}
	return h
}

func TestFinality_GetSafeBlockWithoutLimits(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newHarness(parent.Logger, 0, 0)

			safeBlockNumberAndTime, err := h.service.getFinalitySafeBlockNumber(ctx, RECENT_TIMESTAMP)
			t.Log("safe block number is", safeBlockNumberAndTime)
			require.NoError(t, err, "should not fail")
			require.EqualValues(t, RECENT_BLOCK_NUMBER, safeBlockNumberAndTime.BlockNumber, "should return the recent block number of fake getter")
			require.EqualValues(t, RECENT_TIMESTAMP, safeBlockNumberAndTime.BlockTimeNano, "should return the recent block time of fake getter")
		})
	})
}

func TestFinality_GetSafeBlockWithBlockLimit(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newHarness(parent.Logger, 0, FINALITY_BLOCKS)

			safeBlockNumberAndTime, err := h.service.getFinalitySafeBlockNumber(ctx, RECENT_TIMESTAMP)
			t.Log("safe block number is", safeBlockNumberAndTime)
			require.NoError(t, err, "should not fail")
			require.EqualValues(t, FINALITY_BLOCK_NUMBER, safeBlockNumberAndTime.BlockNumber, "should return 100 blocks before the recent block number of fake getter")
			require.EqualValues(t, FINALITY_BLOCK_TIME, safeBlockNumberAndTime.BlockTimeNano, "should return time of block which is 100 blocks before the recent block time of fake getter")
		})
	})
}

func TestFinality_GetSafeBlockWithTimeLimit(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newHarness(parent.Logger, 200*time.Second, 0)

			safeBlockNumberAndTime, err := h.service.getFinalitySafeBlockNumber(ctx, RECENT_TIMESTAMP)
			t.Log("safe block number is", safeBlockNumberAndTime)
			require.NoError(t, err, "should not fail")
			require.Truef(t, safeBlockNumberAndTime.BlockNumber < RECENT_BLOCK_NUMBER-10, "should return at least 10 blocks before the recent block number of fake getter, but difference is %d", RECENT_BLOCK_NUMBER-safeBlockNumberAndTime.BlockNumber)
		})
	})
}

func TestFinality_GetSafeBlockWithBlockLimit_WhenNotEnoughBlocks(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newHarness(parent.Logger, 0, 2*timestampfinder.FAKE_CLIENT_NUMBER_OF_BLOCKS)

			safeBlockNumber, err := h.service.getFinalitySafeBlockNumber(ctx, RECENT_TIMESTAMP)
			t.Log("safe block number is", safeBlockNumber)
			require.Error(t, err, "should fail because not enough blocks")
		})
	})
}

func TestFinality_VerifySafeBlock(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newHarness(parent.Logger, 0, 100)

			err := h.service.verifyBlockNumberIsFinalitySafe(ctx, RECENT_BLOCK_NUMBER-100, RECENT_TIMESTAMP)
			require.NoError(t, err, "100 difference should be safe")

			err = h.service.verifyBlockNumberIsFinalitySafe(ctx, RECENT_BLOCK_NUMBER-101, RECENT_TIMESTAMP)
			require.NoError(t, err, "101 difference should be safe")

			err = h.service.verifyBlockNumberIsFinalitySafe(ctx, RECENT_BLOCK_NUMBER-99, RECENT_TIMESTAMP)
			require.Error(t, err, "99 difference should not be safe")
		})
	})
}

func TestFinality_GetSafeBlockNeverReturnsNegative(t *testing.T) {
	with.Context(func(ctx context.Context) {
		with.Logging(t, func(parent *with.LoggingHarness) {
			h := newHarness(parent.Logger, 2*time.Minute, 90)

			_, err := h.service.getFinalitySafeBlockNumber(ctx, primitives.TimestampNano(timestampfinder.FAKE_CLIENT_FIRST_TIMESTAMP_SECONDS*time.Second+3*time.Minute))
			require.Error(t, err, "should fail due to negative block number")
		})
	})
}

type finalityConfig struct {
	finalityTimeComponent   time.Duration
	finalityBlocksComponent uint32
}

func (c *finalityConfig) EthereumFinalityTimeComponent() time.Duration {
	return c.finalityTimeComponent
}

func (c *finalityConfig) EthereumFinalityBlocksComponent() uint32 {
	return c.finalityBlocksComponent
}
