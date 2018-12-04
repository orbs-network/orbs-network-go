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
		t.Fatal(err)
	}
	if !hash.Equal(expectedHash) {
		t.Fatalf("Hash invalid, expected %x, got %x", expectedHash, []byte(hash))
	}
}

func TestCalcTxId(t *testing.T) {
	tx := getTransaction()
	txId := digest.CalcTxId(tx)

	// use expected hash and littleEndian encoding of the TS
	// leaving the implementation detail in the test as the encoding part is something the test should 'test'
	expectedHash, err := hex.DecodeString(ExpectedTransactionHashHex)
	if err != nil {
		t.Fatal(err)
	}
	expectedId := make([]byte, 8)
	binary.LittleEndian.PutUint64(expectedId, uint64(tx.Timestamp()))

	expectedId = append(expectedId, expectedHash...)

	if !bytes.Equal(txId, expectedId) {
		t.Fatalf("txid came out wrong, expected %x, got %x", expectedId, txId)
	}

	// extract txid

	txHash, txTimestamp, err := digest.ExtractTxId(txId)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(txHash, expectedHash) {
		t.Fatalf("extracted txHash came out wrong, expected %x, got %x", expectedHash, txHash)
	}

	if !txTimestamp.Equal(tx.Timestamp()) {
		t.Fatalf("extracted txTimestamp came out wrong, expected %s, got %s", tx.Timestamp(), txTimestamp)
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
