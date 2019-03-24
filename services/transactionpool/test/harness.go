// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package test

import (
	"context"
	"fmt"
	"github.com/google/go-cmp/cmp"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/services/transactionpool"
	testKeys "github.com/orbs-network/orbs-network-go/test/crypto/keys"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/gossipmessages"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/gossiptopics"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"testing"
	"time"
)

type harness struct {
	txpool                  services.TransactionPool
	gossip                  *gossiptopics.MockTransactionRelay
	vm                      *services.MockVirtualMachine
	trh                     *handlers.MockTransactionResultsHandler
	lastBlockHeight         primitives.BlockHeight
	lastBlockTimestamp      primitives.TimestampNano
	config                  config.TransactionPoolConfig
	ignoreBlockHeightChecks bool
	logger                  log.BasicLogger
	testOutput              *log.TestOutput
}

var (
	thisNodeKeyPair  = testKeys.EcdsaSecp256K1KeyPairForTests(8)
	otherNodeKeyPair = testKeys.EcdsaSecp256K1KeyPairForTests(9)
)

func (h *harness) expectTransactionsToBeForwarded(sig primitives.EcdsaSecp256K1Sig, transactions ...*protocol.SignedTransaction) {

	h.gossip.When("BroadcastForwardedTransactions", mock.Any, &gossiptopics.ForwardedTransactionsInput{
		Message: &gossipmessages.ForwardedTransactionsMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderNodeAddress: thisNodeKeyPair.NodeAddress(),
				Signature:         sig,
			}).Build(),
			SignedTransactions: transactions,
		},
	}).Return(&gossiptopics.EmptyOutput{}, nil).Times(1)
}

func (h *harness) expectNoTransactionsToBeForwarded() {
	h.gossip.Never("BroadcastForwardedTransactions", mock.Any, mock.Any)
}

func (h *harness) ignoringForwardMessages() {
	h.gossip.When("BroadcastForwardedTransactions", mock.Any, mock.Any).Return(&gossiptopics.EmptyOutput{}, nil).AtLeast(0)
}

func (h *harness) ignoringBlockHeightChecks() {
	h.ignoreBlockHeightChecks = true
}

func (h *harness) addNewTransaction(ctx context.Context, tx *protocol.SignedTransaction) (*services.AddNewTransactionOutput, error) {
	out, err := h.txpool.AddNewTransaction(ctx, &services.AddNewTransactionInput{
		SignedTransaction: tx,
	})

	return out, err
}

func (h *harness) addTransactions(ctx context.Context, txs ...*protocol.SignedTransaction) {
	for _, tx := range txs {
		h.addNewTransaction(ctx, tx)
	}
}

func (h *harness) reportTransactionsAsCommitted(ctx context.Context, transactions ...*protocol.SignedTransaction) (*services.CommitTransactionReceiptsOutput, error) {
	nextBlockHeight := h.lastBlockHeight + 1
	nextTimestamp := primitives.TimestampNano(time.Now().UnixNano())

	out, err := h.txpool.CommitTransactionReceipts(ctx, &services.CommitTransactionReceiptsInput{
		LastCommittedBlockHeight: nextBlockHeight,
		ResultsBlockHeader:       (&protocol.ResultsBlockHeaderBuilder{Timestamp: nextTimestamp, BlockHeight: nextBlockHeight}).Build(), //TODO ResultsBlockHeader is too much info here, awaiting change in proto, see issue #121
		TransactionReceipts:      asReceipts(transactions),
	})

	if err == nil && out.NextDesiredBlockHeight == nextBlockHeight+1 {
		h.lastBlockHeight = nextBlockHeight
		h.lastBlockTimestamp = nextTimestamp
	}

	return out, err
}

func (h *harness) verifyMocks() error {
	if _, err := h.gossip.Verify(); err != nil {
		return err
	}

	if _, err := h.trh.Verify(); err != nil {
		return err
	}

	if _, err := h.vm.Verify(); err != nil {
		return err
	}

	return nil
}

func (h *harness) handleForwardFrom(ctx context.Context, sender *testKeys.TestEcdsaSecp256K1KeyPair, transactions ...*protocol.SignedTransaction) {
	oneBigHash, _, _ := transactionpool.HashTransactions(transactions...)

	sig, err := digest.SignAsNode(sender.PrivateKey(), oneBigHash)
	if err != nil {
		panic(err)
	}

	h.txpool.HandleForwardedTransactions(ctx, &gossiptopics.ForwardedTransactionsInput{
		Message: &gossipmessages.ForwardedTransactionsMessage{
			Sender: (&gossipmessages.SenderSignatureBuilder{
				SenderNodeAddress: sender.NodeAddress(),
				Signature:         sig,
			}).Build(),
			SignedTransactions: transactions,
		},
	})
}

func (h *harness) expectTransactionResultsCallbackFor(transactions ...*protocol.SignedTransaction) {
	h.trh.When("HandleTransactionResults", mock.Any, mock.AnyIf("input has the specified receipts and block height", func(i interface{}) bool {
		input, ok := i.(*handlers.HandleTransactionResultsInput)
		return ok && input.BlockHeight == h.lastBlockHeight+1 && cmp.Equal(input.TransactionReceipts, asReceipts(transactions))
	})).Times(1).Return(&handlers.HandleTransactionResultsOutput{}, nil)
}

func (h *harness) expectTransactionErrorCallbackFor(tx *protocol.SignedTransaction, status protocol.TransactionStatus) {
	txHash := digest.CalcTxHash(tx.Transaction())
	h.trh.When("HandleTransactionError", mock.Any, mock.AnyIf("transaction error matching the given transaction", func(i interface{}) bool {
		tri := i.(*handlers.HandleTransactionErrorInput)
		return tri.Txhash.Equal(txHash) && tri.TransactionStatus == status
	})).Return(&handlers.HandleTransactionErrorOutput{}).Times(1)
}

func (h *harness) ignoringTransactionResults() {
	h.trh.When("HandleTransactionResults", mock.Any, mock.Any)
	h.trh.When("HandleTransactionError", mock.Any, mock.Any)
}

func (h *harness) getTransactionsForOrdering(ctx context.Context, currentBlockHeight primitives.BlockHeight, maxNumOfTransactions uint32) (*services.GetTransactionsForOrderingOutput, error) {
	return h.txpool.GetTransactionsForOrdering(ctx, &services.GetTransactionsForOrderingInput{
		CurrentBlockHeight:      currentBlockHeight,
		PrevBlockTimestamp:      primitives.TimestampNano(time.Now().UnixNano() - 100),
		MaxNumberOfTransactions: maxNumOfTransactions,
	})
}

func (h *harness) failPreOrderCheckFor(failOn func(tx *protocol.SignedTransaction) bool, rejectStatus protocol.TransactionStatus) {
	h.vm.Reset().When("TransactionSetPreOrder", mock.Any, mock.Any).Call(func(ctx context.Context, input *services.TransactionSetPreOrderInput) (*services.TransactionSetPreOrderOutput, error) {
		if !h.ignoreBlockHeightChecks && input.CurrentBlockHeight != h.lastBlockHeight+1 {
			panic(fmt.Sprintf("invalid block height, current is %d and last committed is %d", input.CurrentBlockHeight, h.lastBlockHeight))
		}
		statuses := make([]protocol.TransactionStatus, len(input.SignedTransactions))
		for i, tx := range input.SignedTransactions {
			if failOn(tx) {
				statuses[i] = rejectStatus
			} else {
				statuses[i] = protocol.TRANSACTION_STATUS_PRE_ORDER_VALID
			}
		}
		return &services.TransactionSetPreOrderOutput{
			PreOrderResults: statuses,
		}, nil
	})
}

func (h *harness) passAllPreOrderChecks() {
	h.failPreOrderCheckFor(func(tx *protocol.SignedTransaction) bool {
		return false
	}, protocol.TRANSACTION_STATUS_REJECTED_SMART_CONTRACT_PRE_ORDER)
}

func (h *harness) fastForwardTo(ctx context.Context, height primitives.BlockHeight) {
	h.fastForwardToHeightAndTime(ctx, height, primitives.TimestampNano(time.Now().UnixNano()))
}

func (h *harness) fastForwardToHeightAndTime(ctx context.Context, height primitives.BlockHeight, timestamp primitives.TimestampNano) {
	h.ignoringTransactionResults()
	currentBlock := primitives.BlockHeight(0)
	for currentBlock <= height {
		out, _ := h.txpool.CommitTransactionReceipts(ctx, &services.CommitTransactionReceiptsInput{
			LastCommittedBlockHeight: currentBlock,
			ResultsBlockHeader:       (&protocol.ResultsBlockHeaderBuilder{BlockHeight: currentBlock, Timestamp: timestamp}).Build(),
		})
		currentBlock = out.NextDesiredBlockHeight
	}
	h.lastBlockHeight = height
}

func (h *harness) assumeBlockStorageAtHeight(height primitives.BlockHeight) {
	h.lastBlockHeight = height
	h.lastBlockTimestamp = primitives.TimestampNano(time.Now().UnixNano())
}

func (h *harness) validateTransactionsForOrdering(ctx context.Context, blockHeight primitives.BlockHeight, txs ...*protocol.SignedTransaction) error {
	_, err := h.txpool.ValidateTransactionsForOrdering(ctx, &services.ValidateTransactionsForOrderingInput{
		SignedTransactions:    txs,
		CurrentBlockHeight:    blockHeight,
		CurrentBlockTimestamp: primitives.TimestampNano(time.Now().UnixNano()),
	})

	return err
}

func (h *harness) getTxReceipt(ctx context.Context, tx *protocol.SignedTransaction) (*services.GetCommittedTransactionReceiptOutput, error) {
	return h.txpool.GetCommittedTransactionReceipt(ctx, &services.GetCommittedTransactionReceiptInput{
		Txhash: digest.CalcTxHash(tx.Transaction()),
	})
}

func (h *harness) start(ctx context.Context) *harness {
	service := transactionpool.NewTransactionPool(ctx, h.gossip, h.vm, nil, h.config, h.logger, metric.NewRegistry())
	service.RegisterTransactionResultsHandler(h.trh)
	h.txpool = service
	h.fastForwardTo(ctx, 1)
	return h
}

func (h *harness) allowingErrorsMatching(pattern string) *harness {
	h.testOutput.AllowErrorsMatching(pattern)
	return h
}

const DEFAULT_CONFIG_SIZE_LIMIT = 20 * 1024 * 1024
const DEFAULT_CONFIG_TIME_BETWEEN_EMPTY_BLOCKS_MILLIS = 100

func newHarness(tb testing.TB) *harness {
	return newHarnessWithConfig(tb, DEFAULT_CONFIG_SIZE_LIMIT, DEFAULT_CONFIG_TIME_BETWEEN_EMPTY_BLOCKS_MILLIS*time.Millisecond)
}

func newHarnessWithSizeLimit(tb testing.TB, sizeLimit uint32) *harness {
	return newHarnessWithConfig(tb, sizeLimit, DEFAULT_CONFIG_TIME_BETWEEN_EMPTY_BLOCKS_MILLIS*time.Millisecond)
}

func newHarnessWithInfiniteTimeBetweenEmptyBlocks(tb testing.TB) *harness {
	return newHarnessWithConfig(tb, DEFAULT_CONFIG_SIZE_LIMIT, 1*time.Hour)
}

func newHarnessWithConfig(tb testing.TB, sizeLimit uint32, timeBetweenEmptyBlocks time.Duration) *harness {
	gossip := &gossiptopics.MockTransactionRelay{}
	gossip.When("RegisterTransactionRelayHandler", mock.Any).Return()

	virtualMachine := &services.MockVirtualMachine{}

	cfg := config.ForTransactionPoolTests(sizeLimit, thisNodeKeyPair, timeBetweenEmptyBlocks)
	testOutput := log.NewTestOutput(tb, log.NewHumanReadableFormatter())
	logger := log.GetLogger().WithOutput(testOutput)

	transactionResultHandler := &handlers.MockTransactionResultsHandler{}

	h := &harness{
		gossip:             gossip,
		vm:                 virtualMachine,
		trh:                transactionResultHandler,
		lastBlockTimestamp: primitives.TimestampNano(time.Now().UnixNano()),
		config:             cfg,
		logger:             logger,
		testOutput:         testOutput,
	}

	h.passAllPreOrderChecks()

	return h
}
func asReceipts(transactions transactionpool.Transactions) []*protocol.TransactionReceipt {
	var receipts []*protocol.TransactionReceipt
	for _, tx := range transactions {
		receipts = append(receipts, (&protocol.TransactionReceiptBuilder{
			Txhash: digest.CalcTxHash(tx.Transaction()),
		}).Build())
	}
	return receipts
}
