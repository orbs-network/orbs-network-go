package digest_test

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"testing"
	"time"
)

const (
	ExpectedTransactionHashHex = "4c78c82f8cdd40a923630d38c6b5f48bc257b7a307fb8873495b4b462ef51898"
)

func getTransaction() *protocol.Transaction {
	timeOfTransaction := time.Date(2018, 01, 01, 0, 0, 0, 0, time.UTC)
	tx := builders.TransferTransaction().WithTimestamp(timeOfTransaction).Build()
	return tx.Transaction()
}

func TestCalcTxHash(t *testing.T) {
	// If this test fails it probably means the builder (executed by getTransaction() has changed
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

func BenchmarkCalcTxHash(b *testing.B) {
	b.StopTimer()
	tx := getTransaction()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		digest.CalcTxHash(tx)
	}
}

func BenchmarkCalcTxId(b *testing.B) {
	b.StopTimer()
	tx := getTransaction()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		digest.CalcTxId(tx)
	}
}
