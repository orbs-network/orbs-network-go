package ethereum

import (
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/orbs-network/orbs-network-go/crypto/hash"
	"github.com/orbs-network/orbs-network-go/services/crosschainconnector/ethereum/adapter"
	"github.com/stretchr/testify/require"
	"math/big"
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
          "name": "e",
          "type": "uint256"
        }
      ],
      "name": "MyEvent",
      "type": "event"
    }]`
	//{
	//  "indexed":false,
	//  "name":"d",
	//  "type":"address[]"
	//},
	parsedABI, err := abi.JSON(strings.NewReader(jsonAbi))
	require.NoError(t, err, "failed parsing ABI")
	eventABI := parsedABI.Events["MyEvent"]

	contractAddress := unique32BytesFor(11)
	indexedUint256 := unique32BytesFor(22)
	indexedBytes20 := unique32BytesFor(33)
	nonIndexedAddress := common.BigToAddress(big.NewInt(42))
	nonIndexedUint256 := big.NewInt(42)

	nonIndexedData, err := eventABI.Inputs.NonIndexed().Pack(nonIndexedAddress, nonIndexedUint256)
	require.NoError(t, err, "failed packing non-indexed data")

	log := &adapter.TransactionLog{
		PackedTopics: [][]byte{contractAddress, indexedUint256, indexedBytes20},
		Data:         nonIndexedData,
	}

	res, err := repackEventABIWithTopics(eventABI, log)
	require.NoError(t, err, "failed to run repackEventABIWithTopics")

	require.Equal(t, indexedUint256, get32BytesAtIndex(res, 0), "indexed uint256 value mismatched")
	require.Equal(t, indexedBytes20, get32BytesAtIndex(res, 2), "indexed bytes20 value mismatched")

	unpackedNonIndexedData, err := cloneEventABIWithoutIndexed(eventABI).Inputs.UnpackValues(res)
	require.NoError(t, err, "failed to unpack event inputs")

	require.Equal(t, nonIndexedAddress, unpackedNonIndexedData[1], "non-indexed address value mismatched")
	require.Equal(t, nonIndexedUint256, unpackedNonIndexedData[3], "non-indexed uint256 value mismatched")
}

func unique32BytesFor(num byte) []byte {
	return hash.CalcSha256([]byte{num})
}

func get32BytesAtIndex(buf []byte, index int) []byte {
	from := index * 32
	return buf[from : from+32]
}
