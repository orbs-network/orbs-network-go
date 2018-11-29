package adapter

import (
	"github.com/orbs-network/orbs-network-go/test/builders"
	"testing"
)

func BenchmarkBlockSize(b *testing.B) {
	const KILOBYTE = 1024.0

	emptyBlock := builders.BlockPair().WithTransactions(0).WithReceipts(0).WithStateDiffs(0).Build()
	b.Log("0 tx", float32(sizeOfBlock(emptyBlock))/KILOBYTE)

	hundredTxBlock := builders.BlockPair().WithTransactions(100).WithReceipts(100).WithStateDiffs(100).Build()
	b.Log("100 tx", float32(sizeOfBlock(hundredTxBlock))/KILOBYTE)

	thousandTxBlock := builders.BlockPair().WithTransactions(1000).WithReceipts(1000).WithStateDiffs(1000).Build()
	b.Log("1000 tx", float32(sizeOfBlock(thousandTxBlock))/KILOBYTE)
}
