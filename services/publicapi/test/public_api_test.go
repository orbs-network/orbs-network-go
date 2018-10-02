package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/crypto/digest"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/publicapi"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestSendTransaction_AlreadyCommitted(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, 1*time.Millisecond)

		blockTime := primitives.TimestampNano(time.Now().Nanosecond())
		harness.txpMock.When("AddNewTransaction", mock.Any).Return(&services.AddNewTransactionOutput{
			TransactionStatus:  protocol.TRANSACTION_STATUS_DUPLICATE_TRANSACTION_ALREADY_COMMITTED,
			TransactionReceipt: builders.TransactionReceipt().Build(),
			BlockHeight:        1,
			BlockTimestamp:     blockTime,
		}).Times(1)

		tx, err := harness.papi.SendTransaction(&services.SendTransactionInput{
			ClientRequest: (&client.SendTransactionRequestBuilder{
				SignedTransaction: builders.Transaction().Builder()}).Build(),
		})

		require.NoError(t, err, "error happened when it should not")
		require.NotNil(t, tx, "Send transaction returned nil instead of object")
	})
}

func TestSendTransaction_BlocksUntilTransactionCompletes(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, 1*time.Second)

		blockTime := primitives.TimestampNano(time.Now().Nanosecond())
		txb := builders.Transaction().Builder()
		harness.txpMock.When("AddNewTransaction", mock.Any).Times(1).
			Call(func(input *services.AddNewTransactionInput) (*services.AddNewTransactionOutput, error) {
				go func() {
					time.Sleep(1 * time.Millisecond)
					harness.papi.HandleTransactionResults(&handlers.HandleTransactionResultsInput{
						TransactionReceipts: []*protocol.TransactionReceipt{builders.TransactionReceipt().WithTransaction(txb.Build().Transaction()).Build()},
						BlockHeight:         2,
						Timestamp:           blockTime,
					})
				}()
				return &services.AddNewTransactionOutput{TransactionStatus: protocol.TRANSACTION_STATUS_PENDING}, nil
			})

		tx, err := harness.papi.SendTransaction(&services.SendTransactionInput{
			ClientRequest: (&client.SendTransactionRequestBuilder{
				SignedTransaction: txb,
			}).Build(),
		})

		require.NoError(t, err, "error happened when it should not")
		require.NotNil(t, tx, "Send transaction returned nil instead of object")
	})
}

func TestSendTransaction_BlocksUntilTransactionErrors(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		harness := newPublicApiHarness(ctx, 1*time.Second)

		blockTime := primitives.TimestampNano(time.Now().Nanosecond())
		txb := builders.Transaction().Builder()
		txHash := digest.CalcTxHash(txb.Build().Transaction())

		harness.txpMock.When("AddNewTransaction", mock.Any).Times(1).
			Call(func(input *services.AddNewTransactionInput) (*services.AddNewTransactionOutput, error) {
				go func() {
					time.Sleep(1 * time.Millisecond)
					harness.papi.HandleTransactionError(&handlers.HandleTransactionErrorInput{
						Txhash:            txHash,
						TransactionStatus: protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED,
						BlockHeight:       8,
						BlockTimestamp:    blockTime,
					})
				}()
				return &services.AddNewTransactionOutput{TransactionStatus: protocol.TRANSACTION_STATUS_PENDING}, nil
			})

		tx, err := harness.papi.SendTransaction(&services.SendTransactionInput{
			ClientRequest: (&client.SendTransactionRequestBuilder{
				SignedTransaction: txb,
			}).Build(),
		})

		require.NoError(t, err, "error happened when it should not")
		require.NotNil(t, tx, "Send transaction returned nil instead of object")
		require.Equal(t, protocol.TRANSACTION_STATUS_REJECTED_TIMESTAMP_WINDOW_EXCEEDED, tx.ClientResponse.TransactionStatus(), "got wrong status")
	})
}

type harness struct {
	papi    services.PublicApi
	txpMock *services.MockTransactionPool
}

func newPublicApiHarness(ctx context.Context, txTimeout time.Duration) *harness {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	cfg := newPublicApiConfig(txTimeout)
	txpMock := makeTxMock()
	vmMock := &services.MockVirtualMachine{}
	papi := publicapi.NewPublicApi(ctx, cfg, txpMock, vmMock, logger)
	return &harness{
		papi:    papi,
		txpMock: txpMock,
	}
}

func newPublicApiConfig(txTimeout time.Duration) publicapi.Config {
	cfg := config.EmptyConfig()
	cfg.SetDuration(config.PUBLIC_API_SEND_TRANSACTION_TIMEOUT, txTimeout)

	return cfg
}

func makeTxMock() *services.MockTransactionPool {
	txpMock := &services.MockTransactionPool{}
	txpMock.When("RegisterTransactionResultsHandler", mock.Any).Return(nil)
	return txpMock
}
