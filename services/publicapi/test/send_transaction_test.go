package test

import (
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/services/handlers"
	"github.com/orbs-network/orbs-network-go/services/publicapi"
)

func TestSendTransaction_CallsAddNewTransactionInTxPool(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	txpMock := &services.MockTransactionPool{}
	vmMock := &services.MockVirtualMachine{}
	cfg := config.EmptyConfig()
	papi := publicapi.NewPublicApi(cfg, txpMock, vmMock, logger)

	txpMock.When("AddNewTransaction", mock.Any).Return(&services.AddNewTransactionOutput{}).Times(1)
	papi.SendTransaction(&services.SendTransactionInput{
		ClientRequest: (&client.SendTransactionRequestBuilder{
			SignedTransaction: builders.Transaction().Builder()}).Build(),
	})

	err := test.ConsistentlyVerify(txpMock)
	if err != nil {
		t.Fatal("Did not call AddNewTransaction:", err)
	}
}

func TestSendTransaction_Timeout(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	txpMock := &services.MockTransactionPool{}
	vmMock := &services.MockVirtualMachine{}
	cfg := config.EmptyConfig()
	cfg.SetDuration(config.PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 1*time.Millisecond)

	papi := publicapi.NewPublicApi(cfg, txpMock, vmMock, logger)
	txpMock.When("AddNewTransaction", mock.Any).Return(&services.AddNewTransactionOutput{TransactionStatus: protocol.TRANSACTION_STATUS_PENDING}).Times(1)
	_, err := papi.SendTransaction(&services.SendTransactionInput{
		ClientRequest: (&client.SendTransactionRequestBuilder{
			SignedTransaction: builders.Transaction().Builder()}).Build(),
	})

	require.EqualError(t, err, "timed out waiting for transaction result", "Timeout did not occur")
}

func TestSendTransaction_AlreadyCommitted(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	txpMock := &services.MockTransactionPool{}
	vmMock := &services.MockVirtualMachine{}
	cfg := config.EmptyConfig()
	cfg.SetDuration(config.PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 1*time.Millisecond)

	blocktime := primitives.TimestampNano(time.Now().Nanosecond())
	papi := publicapi.NewPublicApi(cfg, txpMock, vmMock, logger)
	txpMock.When("AddNewTransaction", mock.Any).Return(&services.AddNewTransactionOutput{
		TransactionStatus:  protocol.TRANSACTION_STATUS_DUPLCIATE_TRANSACTION_ALREADY_COMMITTED,
		TransactionReceipt: builders.TransactionReceipt().Build(),
		BlockHeight:        1,
		BlockTimestamp:     blocktime,
	}).Times(1)
	value, err := papi.SendTransaction(&services.SendTransactionInput{
		ClientRequest: (&client.SendTransactionRequestBuilder{
			SignedTransaction: builders.Transaction().Builder()}).Build(),
	})

	require.NoError(t, err, "error happened when it should not")
	require.EqualValues(t, 1, value.ClientResponse.BlockHeight(), "Block height response is wrong")
	require.EqualValues(t, blocktime, value.ClientResponse.BlockTimestamp(), "Block time response is wrong")
	require.EqualValues(t, protocol.TRANSACTION_STATUS_DUPLCIATE_TRANSACTION_ALREADY_COMMITTED, value.ClientResponse.TransactionStatus(), "status response is wrong")
	// TODO test output stuff later
	//require.EqualValues(t, 1, value.ClientResponse.TransactionReceipt()., "Block height response is wrong")
}

// errors ?

func TestSendTransaction_PendingCallsHandler(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	txpMock := &services.MockTransactionPool{}
	vmMock := &services.MockVirtualMachine{}
	cfg := config.EmptyConfig()
	cfg.SetDuration(config.PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 1 * time.Second)

	blocktime := primitives.TimestampNano(time.Now().Nanosecond())
	txb := builders.Transaction().Builder()
	papi := publicapi.NewPublicApi(cfg, txpMock, vmMock, logger)
	txpMock.When("AddNewTransaction", mock.Any).Times(1).
		Call(func(input *services.AddNewTransactionInput) (*services.AddNewTransactionOutput, error){
			go func() {
				time.Sleep(1 * time.Millisecond)
				papi.HandleTransactionResults(&handlers.HandleTransactionResultsInput{
					TransactionReceipts: []*protocol.TransactionReceipt{builders.TransactionReceipt().WithTransaction(txb.Build().Transaction()).Build()},
					BlockHeight:         2,
					Timestamp:           blocktime,
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
	require.EqualValues(t, blocktime, tx.ClientResponse.BlockTimestamp(), "Block time response is wrong")
	require.EqualValues(t, protocol.TRANSACTION_STATUS_COMMITTED, tx.ClientResponse.TransactionStatus(), "status response is wrong")
	// TODO test output stuff later
	//require.EqualValues(t, 1, value.ClientResponse.TransactionReceipt()., "Block height response is wrong")
}


