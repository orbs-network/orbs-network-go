package consensuscontext

import (
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCalculateNewBlockTimestampWithPrevBlockInThePast(t *testing.T) {

	now := primitives.TimestampNano(time.Now().Unix())
	prevBlockTimestamp := now - 1000

	res := calculateNewBlockTimestamp(prevBlockTimestamp, now)
	require.Equal(t, res, now+1, "return 1 nano later than max between now and prev block timestamp")
}

func TestCalculateNewBlockTimestampWithPrevBlockInTheFuture(t *testing.T) {

	now := primitives.TimestampNano(time.Now().Unix())
	prevBlockTimestamp := now + 1000

	res := calculateNewBlockTimestamp(prevBlockTimestamp, now)
	require.Equal(t, res, prevBlockTimestamp+1, "return 1 nano later than max between now and prev block timestamp")
}
