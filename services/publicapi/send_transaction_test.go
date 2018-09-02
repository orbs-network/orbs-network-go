package publicapi

import (
	"context"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestSendTransaction_Timeout(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	txpMock := &services.MockTransactionPool{}
	cfg := config.EmptyConfig()
	cfg.SetDuration(config.PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 1*time.Millisecond)

	txpMock.When("RegisterTransactionResultsHandler", mock.Any).Return(nil)
	ctx, cancel := context.WithCancel(context.Background())
	papi := NewPublicApi(ctx, cfg, txpMock, nil, logger)
	txpMock.When("AddNewTransaction", mock.Any).Return(&services.AddNewTransactionOutput{TransactionStatus: protocol.TRANSACTION_STATUS_PENDING}).Times(1)
	_, err := papi.SendTransaction(&services.SendTransactionInput{
		ClientRequest: (&client.SendTransactionRequestBuilder{
			SignedTransaction: builders.Transaction().Builder()}).Build(),
	})

	require.EqualError(t, err, "timed out waiting for transaction result", "Timeout did not occur")
	cancel() // TODO wait for termination
}
