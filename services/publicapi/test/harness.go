package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/publicapi"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"os"
	"time"
)

type harness struct {
	papi    services.PublicApi
	txpMock *services.MockTransactionPool
	bksMock *services.MockBlockStorage
}

func newPublicApiHarness(ctx context.Context, txTimeout time.Duration) *harness {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	cfg := newPublicApiConfig(txTimeout)
	txpMock := makeTxMock()
	vmMock := &services.MockVirtualMachine{}
	bksMock := &services.MockBlockStorage{}
	papi := publicapi.NewPublicApi(ctx, cfg, txpMock, vmMock, bksMock, logger)
	return &harness{
		papi:    papi,
		txpMock: txpMock,
		bksMock: bksMock,
	}
}

func newPublicApiConfig(txTimeout time.Duration) publicapi.Config {
	cfg := config.EmptyConfig()
	cfg.SetDuration(config.PUBLIC_API_SEND_TRANSACTION_TIMEOUT, txTimeout)
	cfg.SetUint32(config.VIRTUAL_CHAIN_ID, uint32(builders.DEFAULT_TEST_VIRTUAL_CHAIN_ID))

	return cfg
}

func makeTxMock() *services.MockTransactionPool {
	txpMock := &services.MockTransactionPool{}
	txpMock.When("RegisterTransactionResultsHandler", mock.Any).Return(nil)
	return txpMock
}

func (h *harness) onAddNewTransaction(f func()) {
	h.txpMock.When("AddNewTransaction", mock.Any).Times(1).
		Call(func(input *services.AddNewTransactionInput) (*services.AddNewTransactionOutput, error) {
			go func() {
				time.Sleep(1 * time.Millisecond)
				f()
			}()
			return &services.AddNewTransactionOutput{TransactionStatus: protocol.TRANSACTION_STATUS_PENDING}, nil
		})
}

func (h *harness) transactionIsPendingInPool() {
	h.txpMock.When("GetCommittedTransactionReceipt", mock.Any).Return(&services.GetCommittedTransactionReceiptOutput{
		TransactionStatus: protocol.TRANSACTION_STATUS_PENDING,
	}).Times(1)
}

func (h *harness) transactionIsNotInPool() {
	h.txpMock.When("GetCommittedTransactionReceipt", mock.Any).Return(&services.GetCommittedTransactionReceiptOutput{
		TransactionStatus: protocol.TRANSACTION_STATUS_NO_RECORD_FOUND,
	}).Times(1)
}

func (h *harness) transactionIsCommitedInPool() {
	h.txpMock.When("GetCommittedTransactionReceipt", mock.Any).Return(&services.GetCommittedTransactionReceiptOutput{
		TransactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
		TransactionReceipt: builders.TransactionReceipt().Build(),
	}).Times(1)
}

func (h *harness) transactionIsInBlockStorage() {
	h.bksMock.When("GetTransactionReceipt", mock.Any).Return(
		&services.GetTransactionReceiptOutput{
			TransactionReceipt: builders.TransactionReceipt().Build(),
			BlockHeight:        0,
			BlockTimestamp:     0,
		}).Times(1)
}
