// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package consensuscontext

import (
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCalculateNewBlockTimestampWithPrevBlockInThePast(t *testing.T) {

	now := primitives.TimestampNano(time.Now().Unix())
	prevBlockTimestamp := now - 1000

	res := digest.CalcNewBlockTimestamp(prevBlockTimestamp, now)
	require.Equal(t, res, now, "return 1 nano later than max between now and prev block timestamp")
}

func TestCalculateNewBlockTimestampWithPrevBlockInTheFuture(t *testing.T) {

	now := primitives.TimestampNano(time.Now().Unix())
	prevBlockTimestamp := now + 1000

	res := digest.CalcNewBlockTimestamp(prevBlockTimestamp, now)
	require.Equal(t, res, prevBlockTimestamp+1, "return 1 nano later than max between now and prev block timestamp")
}
