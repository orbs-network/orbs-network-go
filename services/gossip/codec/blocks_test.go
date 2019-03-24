// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package codec

import (
	"github.com/google/go-cmp/cmp"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestBlockPair(t *testing.T) {
	tests := []struct {
		name      string
		origin    *protocol.BlockPairContainer
		encodeErr bool
		decodeErr bool
	}{
		{
			"MissingTxAndRx",
			builders.CorruptBlockPair().WithMissingTransactionsBlock().WithMissingResultsBlock().Build(),
			true,
			false,
		},
		{
			"EmptyTxAndRx",
			builders.CorruptBlockPair().WithEmptyTransactionsBlock().WithEmptyResultsBlock().Build(),
			true,
			false,
		},
		{
			"EmptyBlock",
			builders.BlockPair().WithTransactions(0).WithReceipts(0).WithStateDiffs(0).Build(),
			false,
			false,
		},
		{
			"NormalBlock",
			builders.BlockPair().WithTransactions(3).WithReceipts(3).WithStateDiffs(2).Build(),
			false,
			false,
		},
		{
			"CorruptNumTx",
			builders.BlockPair().WithCorruptNumTransactions(25).Build(),
			false,
			true,
		},
		{
			"CorruptNumReceipts",
			builders.BlockPair().WithCorruptNumReceipts(34).Build(),
			false,
			true,
		},
		{
			"CorruptNumStateDiffs",
			builders.BlockPair().WithCorruptNumStateDiffs(29).Build(),
			false,
			true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// encode
			payloads, err := EncodeBlockPair(tt.origin)
			if tt.encodeErr {
				require.Error(t, err, "encode should return an error")
				return
			}
			require.NoError(t, err, "encode should not return an error")

			// decode
			res, err := DecodeBlockPair(payloads)
			if tt.decodeErr {
				require.Error(t, err, "decode should return an error")
				return
			}
			require.NoError(t, err, "decode should not return an error")
			test.RequireCmpEqual(t, tt.origin, res)
		})
	}
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

func TestMultipleBlockPairs(t *testing.T) {
	for _, tt := range multipleBlockPairsTable {
		payloads, err := EncodeBlockPairs(tt.origin)
		if tt.encodeErr != (err != nil) {
			t.Fatalf("Expected encode error to be %v but got: %s", tt.encodeErr, err)
		}
		if err != nil {
			continue
		}
		res, err := DecodeBlockPairs(payloads)
		if tt.decodeErr != (err != nil) {
			t.Fatalf("Expected decode error to be %v but got: %s", tt.decodeErr, err)
		}
		if err != nil {
			continue
		}
		if !cmp.Equal(res, tt.origin) {
			t.Fatalf("Result and origin are different: %v", cmp.Diff(res, tt.origin))
		}
	}
}
