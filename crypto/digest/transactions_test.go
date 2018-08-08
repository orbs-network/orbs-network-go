package digest_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"testing"
	"time"
)

const (
	ExpectedTransactionHashHex = "d54302c7a373ef3f5064eb42175e84fa174fb2a23a22d1e0ebe50e292657e6a4"
)

func getTransaction() *protocol.Transaction {
	timeOfTransaction := primitives.TimestampNano(time.Date(2018, 01, 01, 0, 0, 0, 0, time.UTC).UnixNano())
	tx := builders.TransferTransaction().WithAmount(10).WithTimestamp(timeOfTransaction).Build()
	return tx.Transaction()
}

func TestCalcTxHash(t *testing.T) {
	tx := getTransaction()
	hash := digest.CalcTxHash(tx)
	expectedHash, err := hex.DecodeString(ExpectedTransactionHashHex)
	if err != nil {
		t.Error(err)
	}
	if !hash.Equal(expectedHash) {
		t.Errorf("Hash invalid, expected %x, got %x", expectedHash, []byte(hash))
	}
}

func TestCalcTxId(t *testing.T) {
	tx := getTransaction()
	txId := digest.CalcTxId(tx)

	// use expected hash and littleEndian encoding of the TS
	// leaving the implementation detail in the test as the encoding part is something the test should 'test'
	expectedHash, err := hex.DecodeString(ExpectedTransactionHashHex)
	if err != nil {
		t.Error(err)
	}
	expectedId := make([]byte, 8)
	binary.LittleEndian.PutUint64(expectedId, uint64(tx.Timestamp()))

	expectedId = append(expectedId, expectedHash...)

	if !bytes.Equal(txId, expectedId) {
		t.Errorf("txid came out wrong, expected %x, got %x", expectedId, txId)
	}

}
