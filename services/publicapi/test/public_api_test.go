package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
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
		logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
		txpMock := makeTxMock()
		vmMock := &services.MockVirtualMachine{}
		cfg := config.EmptyConfig()
		cfg.SetDuration(config.PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 1*time.Millisecond)
		papi := publicapi.NewPublicApi(ctx, cfg, txpMock, vmMock, logger)

		blockTime := primitives.TimestampNano(time.Now().Nanosecond())
		txpMock.When("AddNewTransaction", mock.Any).Return(&services.AddNewTransactionOutput{
			TransactionStatus:  protocol.TRANSACTION_STATUS_DUPLCIATE_TRANSACTION_ALREADY_COMMITTED,
			TransactionReceipt: builders.TransactionReceipt().Build(),
			BlockHeight:        1,
			BlockTimestamp:     blockTime,
		}).Times(1)

		tx, err := papi.SendTransaction(&services.SendTransactionInput{
			ClientRequest: (&client.SendTransactionRequestBuilder{
				SignedTransaction: builders.Transaction().Builder()}).Build(),
		})

		require.NoError(t, err, "error happened when it should not")
		require.NotNil(t, tx, "Send transaction returned nil instead of object")
	})
}

func TestSendTransaction_BlocksUntilTransactionCompletes(t *testing.T) {
	test.WithContext(func(ctx context.Context) {
		logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
		txpMock := makeTxMock()
		vmMock := &services.MockVirtualMachine{}
		cfg := config.EmptyConfig()
		cfg.SetDuration(config.PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 1*time.Second)
		papi := publicapi.NewPublicApi(ctx, cfg, txpMock, vmMock, logger)

		blockTime := primitives.TimestampNano(time.Now().Nanosecond())
		txb := builders.Transaction().Builder()
		txpMock.When("AddNewTransaction", mock.Any).Times(1).
			Call(func(input *services.AddNewTransactionInput) (*services.AddNewTransactionOutput, error) {
				go func() {
					time.Sleep(1 * time.Millisecond)
					papi.HandleTransactionResults(&handlers.HandleTransactionResultsInput{
						TransactionReceipts: []*protocol.TransactionReceipt{builders.TransactionReceipt().WithTransaction(txb.Build().Transaction()).Build()},
						BlockHeight:         2,
						Timestamp:           blockTime,
					})
				}()
				return &services.AddNewTransactionOutput{TransactionStatus: protocol.TRANSACTION_STATUS_PENDING}, nil
			})

		tx, err := papi.SendTransaction(&services.SendTransactionInput{
			ClientRequest: (&client.SendTransactionRequestBuilder{
				SignedTransaction: txb,
			}).Build(),
		})

		require.NoError(t, err, "error happened when it should not")
		require.NotNil(t, tx, "Send transaction returned nil instead of object")
	})
}

func makeTxMock() *services.MockTransactionPool {
	txpMock := &services.MockTransactionPool{}
	txpMock.When("RegisterTransactionResultsHandler", mock.Any).Return(nil)
	return txpMock
}
