package publicapi

import (
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestPublicApiSendTx_PrepareResponse(t *testing.T) {
	blockTime := primitives.TimestampNano(time.Now().Nanosecond())
	receipt := builders.TransactionReceipt().WithRandomHash().Build()

	response := toSendTxOutput(&txResponse{
		transactionStatus:  protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED,
		transactionReceipt: receipt,
		blockHeight:        126,
		blockTimestamp:     blockTime,
	})

	test.RequireCmpEqual(t, receipt, response.ClientResponse.TransactionReceipt(), "Transaction receipt is not equal")
	require.EqualValues(t, 126, response.ClientResponse.BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, blockTime, response.ClientResponse.BlockTimestamp(), "Block time response is wrong")
	require.EqualValues(t, protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED, response.ClientResponse.TransactionStatus(), "status response is wrong")
}

func TestPublicApiSendTx_PrepareResponseNilReceipt(t *testing.T) {
	blockTime := primitives.TimestampNano(time.Now().Nanosecond())

	response := toSendTxOutput(&txResponse{
		transactionStatus:  protocol.TRANSACTION_STATUS_REJECTED_CONGESTION,
		transactionReceipt: nil,
		blockHeight:        8,
		blockTimestamp:     blockTime,
	})

	test.RequireCmpEqual(t, (*protocol.TransactionReceiptBuilder)(nil).Build(), response.ClientResponse.TransactionReceipt(), "Transaction receipt is not equal")
	require.Equal(t, 0, len(response.ClientResponse.TransactionReceipt().Raw()), "Transaction receipt is not equal") // different way
	require.EqualValues(t, 8, response.ClientResponse.BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, blockTime, response.ClientResponse.BlockTimestamp(), "Block time response is wrong")
	require.EqualValues(t, protocol.TRANSACTION_STATUS_REJECTED_CONGESTION, response.ClientResponse.TransactionStatus(), "status response is wrong")
}
