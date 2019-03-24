// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package ethereum

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/timestampfinder"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

const RECENT_TIMESTAMP = primitives.TimestampNano(1505735343000000000)
const RECENT_BLOCK_NUMBER_OF_FAKE_GETTER = 938874

type harness struct {
	cfg    *finalityConfig
	finder timestampfinder.TimestampFinder
}

func newHarness(t testing.TB, fct time.Duration, fbc uint32) *harness {
	logger := log.DefaultTestingLogger(t)
	cfg := &finalityConfig{fct, fbc}

	btg := timestampfinder.NewFakeBlockTimeGetter(logger)
	finder := timestampfinder.NewTimestampFinder(btg, logger, metric.NewRegistry())

	h := &harness{
		cfg:    cfg,
		finder: finder,
	}
	return h
}

func TestFinality_GetSafeBlockWithoutLimits(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t, 0, 0)

		safeBlockNumber, err := getFinalitySafeBlockNumber(ctx, RECENT_TIMESTAMP, h.finder, h.cfg)
		t.Log("safe block number is", safeBlockNumber)
		require.NoError(t, err, "should not fail")
		require.EqualValues(t, RECENT_BLOCK_NUMBER_OF_FAKE_GETTER, safeBlockNumber.Uint64(), "should return the recent block number of fake getter")
	})
}

func TestFinality_GetSafeBlockWithBlockLimit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t, 0, 100)

		safeBlockNumber, err := getFinalitySafeBlockNumber(ctx, RECENT_TIMESTAMP, h.finder, h.cfg)
		t.Log("safe block number is", safeBlockNumber)
		require.NoError(t, err, "should not fail")
		require.EqualValues(t, RECENT_BLOCK_NUMBER_OF_FAKE_GETTER-100, safeBlockNumber.Uint64(), "should return 100 blocks before the recent block number of fake getter")
	})
}

func TestFinality_GetSafeBlockWithTimeLimit(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t, 200*time.Second, 0)

		safeBlockNumber, err := getFinalitySafeBlockNumber(ctx, RECENT_TIMESTAMP, h.finder, h.cfg)
		t.Log("safe block number is", safeBlockNumber)
		require.NoError(t, err, "should not fail")
		require.Truef(t, safeBlockNumber.Uint64() < RECENT_BLOCK_NUMBER_OF_FAKE_GETTER-10, "should return at least 10 blocks before the recent block number of fake getter, but difference is %d", RECENT_BLOCK_NUMBER_OF_FAKE_GETTER-safeBlockNumber.Uint64())
	})
}

func TestFinality_GetSafeBlockWithBlockLimit_WhenNotEnoughBlocks(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t, 0, 2*timestampfinder.FAKE_CLIENT_NUMBER_OF_BLOCKS)

		safeBlockNumber, err := getFinalitySafeBlockNumber(ctx, RECENT_TIMESTAMP, h.finder, h.cfg)
		t.Log("safe block number is", safeBlockNumber)
		require.Error(t, err, "should fail because not enough blocks")
	})
}

func TestFinality_VerifySafeBlock(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t, 0, 100)

		err := verifyBlockNumberIsFinalitySafe(ctx, RECENT_BLOCK_NUMBER_OF_FAKE_GETTER-100, RECENT_TIMESTAMP, h.finder, h.cfg)
		require.NoError(t, err, "100 difference should be safe")

		err = verifyBlockNumberIsFinalitySafe(ctx, RECENT_BLOCK_NUMBER_OF_FAKE_GETTER-101, RECENT_TIMESTAMP, h.finder, h.cfg)
		require.NoError(t, err, "101 difference should be safe")

		err = verifyBlockNumberIsFinalitySafe(ctx, RECENT_BLOCK_NUMBER_OF_FAKE_GETTER-99, RECENT_TIMESTAMP, h.finder, h.cfg)
		require.Error(t, err, "99 difference should not be safe")
	})
}

func TestFinality_GetSafeBlockNeverReturnsNegative(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		h := newHarness(t, 2*time.Minute, 90)

		_, err := getFinalitySafeBlockNumber(ctx, primitives.TimestampNano(timestampfinder.FAKE_CLIENT_FIRST_TIMESTAMP_SECONDS*time.Second+3*time.Minute), h.finder, h.cfg)
		require.Error(t, err, "should fail due to negative block number")
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
