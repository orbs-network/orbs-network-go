// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package timestampfinder

import (
	"context"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/stretchr/testify/require"
	"math/big"
	"testing"
	"time"
)

// this test exists to make sure that the fake timestamp/block pairs remain constant, as other tests in the system (such as header_by_timestamp_finder_test.go) rely on these constant numbers
func TestFakeBlockHeaderFetcherRawData(t *testing.T) {
	btg := NewFakeBlockTimeGetter(log.DefaultTestingLogger(t))

	require.EqualValues(t, FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED_SECONDS, btg.data[FAKE_CLIENT_NUMBER_OF_BLOCKS], "expected getter last block to be of specific ts")
}

func TestFakeBlockHeaderFetcherOfLatest(t *testing.T) {
	btg := NewFakeBlockTimeGetter(log.DefaultTestingLogger(t))

	b, err := btg.GetTimestampForLatestBlock(context.Background())
	require.NoError(t, err, "should not fail getting 'latest' from fake db")
	require.EqualValues(t, secondsToNano(FAKE_CLIENT_LAST_TIMESTAMP_EXPECTED_SECONDS), b.BlockTimeNano, "expected getter last block to be of specific ts")
	require.EqualValues(t, FAKE_CLIENT_NUMBER_OF_BLOCKS, b.BlockNumber, "expected last block of constant number")
}

func TestFakeBlockLatency(t *testing.T) {
	btg := NewFakeBlockTimeGetter(log.DefaultTestingLogger(t)).WithLatency(10 * time.Millisecond)
	start := time.Now()

	btg.GetTimestampForBlockNumber(context.Background(), big.NewInt(15))

	d := time.Since(start)
	require.True(t, d > 10*time.Millisecond, "expected some latency when getting the block")
}
