package ethereum

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

// TODO(v1): add tests for events with tuples (both indexed and not)
func TestRepackEventABIWithTopics(t *testing.T) {
	jsonAbi := `[{
      "anonymous": false,
      "inputs": [
        {
          "indexed": true,
          "name": "a",
          "type": "uint256"
        },
        {
          "indexed": false,
          "name": "b",
          "type": "address"
        },
        {
          "indexed": true,
          "name": "c",
          "type": "bytes20"
        },
        {
          "indexed": false,
          "name": "d",
          "type": "uint256"
        }
      ],
      "name": "MyEvent",
      "type": "event"
    }]`

	parsedABI, err := abi.JSON(strings.NewReader(jsonAbi))
	require.NoError(t, err, "failed parsing ABI")

	log := &adapter.TransactionLog{
		PackedTopics: [][]byte{unique32BytesFor(11), unique32BytesFor(22), unique32BytesFor(33)},
		Data:         append(unique32BytesFor(44), unique32BytesFor(55)...),
	}

	res, err := repackEventABIWithTopics(parsedABI.Events["MyEvent"], log)
	require.NoError(t, err, "failed to run repackEventABIWithTopics")

	require.Equal(t, unique32BytesFor(22), get32BytesAtIndex(res, 0))
	require.Equal(t, unique32BytesFor(44), get32BytesAtIndex(res, 1))
	require.Equal(t, unique32BytesFor(33), get32BytesAtIndex(res, 2))
	require.Equal(t, unique32BytesFor(55), get32BytesAtIndex(res, 3))
}

func unique32BytesFor(num byte) []byte {
	return hash.CalcSha256([]byte{num})
}

func get32BytesAtIndex(buf []byte, index int) []byte {
	from := index * 32
	return buf[from : from+32]
}
