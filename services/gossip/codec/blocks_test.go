package codec

import (
	"github.com/google/go-cmp/cmp"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"testing"
)

var blockPairTable = []struct {
	origin    *protocol.BlockPairContainer
	encodeErr bool
	decodeErr bool
}{
	{builders.CorruptBlockPair().WithMissingTransactionsBlock().WithMissingResultsBlock().Build(), true, false},
	{builders.CorruptBlockPair().WithEmptyTransactionsBlock().WithEmptyResultsBlock().Build(), true, false},
	{builders.BlockPair().WithTransactions(0).WithReceipts(0).WithStateDiffs(0).Build(), false, false},
	{builders.BlockPair().WithTransactions(3).WithReceipts(3).WithStateDiffs(2).Build(), false, false},
	{builders.BlockPair().WithCorruptNumTransactions(25).Build(), false, true},
	{builders.BlockPair().WithCorruptNumReceipts(34).Build(), false, true},
	{builders.BlockPair().WithCorruptNumStateDiffs(29).Build(), false, true},
}

var multipleBlockPairsTable = []struct {
	origin    []*protocol.BlockPairContainer
	encodeErr bool
	decodeErr bool
}{
	{
		[]*protocol.BlockPairContainer{
			builders.BlockPair().WithTransactions(5).WithReceipts(5).WithStateDiffs(3).Build(),
			builders.CorruptBlockPair().WithMissingTransactionsBlock().WithMissingResultsBlock().Build(),
		}, true, false,
	},
	{
		[]*protocol.BlockPairContainer{
			builders.BlockPair().WithTransactions(5).WithReceipts(5).WithStateDiffs(3).Build(),
			builders.CorruptBlockPair().WithEmptyTransactionsBlock().WithEmptyResultsBlock().Build(),
		},
		true, false,
	},
	{
		[]*protocol.BlockPairContainer{
			builders.BlockPair().WithTransactions(5).WithReceipts(5).WithStateDiffs(3).Build(),
			builders.BlockPair().WithTransactions(0).WithReceipts(0).WithStateDiffs(0).Build(),
		},
		false, false,
	},
	{
		[]*protocol.BlockPairContainer{
			builders.BlockPair().WithTransactions(5).WithReceipts(5).WithStateDiffs(3).Build(),
			builders.BlockPair().WithTransactions(3).WithReceipts(3).WithStateDiffs(2).Build(),
		},
		false, false,
	},
	{
		[]*protocol.BlockPairContainer{
			builders.BlockPair().WithTransactions(5).WithReceipts(5).WithStateDiffs(3).Build(),
			builders.BlockPair().WithCorruptNumTransactions(25).Build(),
		},
		false, true,
	},
	{
		[]*protocol.BlockPairContainer{
			builders.BlockPair().WithTransactions(5).WithReceipts(5).WithStateDiffs(3).Build(),
			builders.BlockPair().WithCorruptNumReceipts(34).Build(),
		},
		false, true,
	},
	{
		[]*protocol.BlockPairContainer{
			builders.BlockPair().WithTransactions(5).WithReceipts(5).WithStateDiffs(3).Build(),
			builders.BlockPair().WithCorruptNumStateDiffs(29).Build(),
		},
		false, true,
	},
}

func TestBlockPair(t *testing.T) {
	for _, tt := range blockPairTable {
		payloads, err := EncodeBlockPair(tt.origin)
		if tt.encodeErr != (err != nil) {
			t.Fatalf("Expected encode error to be %v but got: %v", tt.encodeErr, err)
		}
		if err != nil {
			continue
		}
		res, err := DecodeBlockPair(payloads)
		if tt.decodeErr != (err != nil) {
			t.Fatalf("Expected decode error to be %v but got: %v", tt.decodeErr, err)
		}
		if err != nil {
			continue
		}
		if !cmp.Equal(res, tt.origin) {
			t.Fatalf("Result and origin are different: %v", cmp.Diff(res, tt.origin))
		}
	}
}

func TestMultipleBlockPairs(t *testing.T) {
	for _, tt := range multipleBlockPairsTable {
		payloads, err := EncodeBlockPairs(tt.origin)
		if tt.encodeErr != (err != nil) {
			t.Fatalf("Expected encode error to be %v but got: %v", tt.encodeErr, err)
		}
		if err != nil {
			continue
		}
		res, err := DecodeBlockPairs(payloads)
		if tt.decodeErr != (err != nil) {
			t.Fatalf("Expected decode error to be %v but got: %v", tt.decodeErr, err)
		}
		if err != nil {
			continue
		}
		if !cmp.Equal(res, tt.origin) {
			t.Fatalf("Result and origin are different: %v", cmp.Diff(res, tt.origin))
		}
	}
}
