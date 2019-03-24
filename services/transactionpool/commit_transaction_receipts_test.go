// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package transactionpool

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/rand"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"testing"
)

type committedRecord struct {
	receipt     *protocol.TransactionReceipt
	blockHeight primitives.BlockHeight
	blockTime   primitives.TimestampNano
}

type fakeAdderRemover struct {
	txs       map[string]*primitives.NodeAddress
	committed []*committedRecord
}

func (a *fakeAdderRemover) remove(ctx context.Context, txHash primitives.Sha256, removalReason protocol.TransactionStatus) *primitives.NodeAddress {
	return a.txs[txHash.KeyForMap()]
}

func (a *fakeAdderRemover) add(receipt *protocol.TransactionReceipt, blockHeight primitives.BlockHeight, blockTs primitives.TimestampNano) {
	a.committed = append(a.committed, &committedRecord{receipt: receipt, blockHeight: blockHeight, blockTime: blockTs})
}

func (a *fakeAdderRemover) originFor(receipt *protocol.TransactionReceipt, address primitives.NodeAddress) {
	a.txs[receipt.Txhash().KeyForMap()] = &address
}

func newFake() *fakeAdderRemover {
	return &fakeAdderRemover{txs: make(map[string]*primitives.NodeAddress)}
}

func TestCommitTransactionReceipts_EnqueuesOnlyNodesTransactionsForNotification(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		rnd := rand.NewControlledRand(t)
		n1 := []byte{0x1}
		n2 := []byte{0x2}

		fake := newFake()
		c := &committer{
			adder:       fake,
			remover:     fake,
			logger:      log.DefaultTestingLogger(t),
			nodeAddress: n1,
		}

		tx1 := builders.TransactionReceipt().WithRandomHash(rnd).Build()
		tx2 := builders.TransactionReceipt().WithRandomHash(rnd).Build()
		tx3 := builders.TransactionReceipt().WithRandomHash(rnd).Build()

		fake.originFor(tx1, n1)
		fake.originFor(tx2, n1)
		fake.originFor(tx3, n2)

		c.commit(ctx, tx1, tx2, tx3)

		require.Contains(t, c.myReceipts, tx1, "a tx from n1 gateway was not enqueued for reporting")
		require.Contains(t, c.myReceipts, tx2, "a tx from n1 gateway was not enqueued for reporting")
		require.NotContains(t, c.myReceipts, tx3, "a tx from n2 gateway was not enqueued for reporting")
	})
}

func TestCommitTransactionReceipts_AddsToCommittedPoolWithCorrectExpirationTime(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		fake := newFake()
		c := &committer{
			adder:       fake,
			remover:     fake,
			logger:      log.DefaultTestingLogger(t),
			blockTime:   1,
			blockHeight: 2,
		}

		tx1 := builders.TransactionReceipt().Build()

		c.commit(ctx, tx1)

		require.Len(t, fake.committed, 1, "did not add anything to committed pool")
		require.Equal(t, tx1, fake.committed[0].receipt, "did not add the tx receipt to committed pool")
		require.Equal(t, c.blockHeight, fake.committed[0].blockHeight, "committed transaction has wrong block height")
		require.Equal(t, c.blockTime, fake.committed[0].blockTime, "committed transaction has wrong block timestamp")
	})
}

func TestCommitTransactionReceipts_NotifiesResultHandlers(t *testing.T) {
	test.WithContext(func(ctx context.Context) {

		rnd := rand.NewControlledRand(t)
		tx1 := builders.TransactionReceipt().WithRandomHash(rnd).Build()
		tx2 := builders.TransactionReceipt().WithRandomHash(rnd).Build()

		c := &committer{
			logger:      log.DefaultTestingLogger(t),
			myReceipts:  []*protocol.TransactionReceipt{tx1, tx2},
			blockHeight: 1,
			blockTime:   2,
		}

		handler := &handlers.MockTransactionResultsHandler{}
		handler.When("HandleTransactionResults", mock.Any, &handlers.HandleTransactionResultsInput{
			TransactionReceipts: c.myReceipts,
			BlockHeight:         c.blockHeight,
			Timestamp:           c.blockTime,
		}).Times(1)

		c.notify(ctx, handler)

		_, err := handler.Verify()
		require.NoError(t, err, "handler was not invoked as expected")
	})
}
