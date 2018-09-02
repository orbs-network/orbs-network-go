package test

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/services/publicapi"
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
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	txpMock := &services.MockTransactionPool{}
	vmMock := &services.MockVirtualMachine{}
	cfg := config.EmptyConfig()
	cfg.SetDuration(config.PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 1*time.Millisecond)

	blockTime := primitives.TimestampNano(time.Now().Nanosecond())
	txpMock.When("RegisterTransactionResultsHandler", mock.Any).Return(nil)
	ctx, cancel := context.WithCancel(context.Background())
	papi := publicapi.NewPublicApi(ctx, cfg, txpMock, vmMock, logger)
	txpMock.When("AddNewTransaction", mock.Any).Return(&services.AddNewTransactionOutput{
		TransactionStatus:  protocol.TRANSACTION_STATUS_DUPLCIATE_TRANSACTION_ALREADY_COMMITTED,
		TransactionReceipt: builders.TransactionReceipt().Build(),
		BlockHeight:        1,
		BlockTimestamp:     blockTime,
	}).Times(1)
	value, err := papi.SendTransaction(&services.SendTransactionInput{
		ClientRequest: (&client.SendTransactionRequestBuilder{
			SignedTransaction: builders.Transaction().Builder()}).Build(),
	})

	require.NoError(t, err, "error happened when it should not")
	require.EqualValues(t, 1, value.ClientResponse.BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, blockTime, value.ClientResponse.BlockTimestamp(), "Block time response is wrong")
	require.EqualValues(t, protocol.TRANSACTION_STATUS_DUPLCIATE_TRANSACTION_ALREADY_COMMITTED, value.ClientResponse.TransactionStatus(), "status response is wrong")
	// TODO test output stuff later
	//require.EqualValues(t, 1, value.ClientResponse.TransactionReceipt()., "Block height response is wrong")

	cancel() // TODO wait for termination
}

func TestSendTransaction_BlocksUntilTransactionCompletes(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	txpMock := &services.MockTransactionPool{}
	vmMock := &services.MockVirtualMachine{}
	cfg := config.EmptyConfig()
	cfg.SetDuration(config.PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 1*time.Second)

	blockTime := primitives.TimestampNano(time.Now().Nanosecond())
	txb := builders.Transaction().Builder()
	txpMock.When("RegisterTransactionResultsHandler", mock.Any).Return(nil)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	papi := publicapi.NewPublicApi(ctx, cfg, txpMock, vmMock, logger)

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
	require.EqualValues(t, 2, tx.ClientResponse.BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, blockTime, tx.ClientResponse.BlockTimestamp(), "Block time response is wrong")
	require.EqualValues(t, protocol.TRANSACTION_STATUS_COMMITTED, tx.ClientResponse.TransactionStatus(), "status response is wrong")
}
