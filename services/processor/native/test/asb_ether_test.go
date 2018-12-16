package test

import (
	. "github.com/orbs-network/orbs-contract-sdk/go/fake"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTransferIn_AlloGood(t *testing.T) {

	hexEncodedEthTxHash := "abcd"
	address := "a"
	abi := "b"
	txid := "c"
	eventName := "d"

	InServiceScope(AnAddress(), func(m Mockery) {

		m.MockEthereumLog(address, abi, txid, eventName, func(out interface{}) {
			out.(*event).value = "foo"
		})

		v := readValueFromLog(address, abi, txid, eventName)

		require.Equal(t, "foo", v, "did not get expected value from log")
	})

}
