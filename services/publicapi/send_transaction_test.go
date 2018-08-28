package publicapi

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
)

func TestSendTransaction_CallsAddNewTransactionInTxPool(t *testing.T) {
	log := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	txpMock := &services.MockTransactionPool{}
	vmMock := &services.MockVirtualMachine{}
	cfg := config.EmptyConfig()
	papi := NewPublicApi(cfg, txpMock, vmMock, log)

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
	log := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	txpMock := &services.MockTransactionPool{}
	vmMock := &services.MockVirtualMachine{}
	cfg := config.EmptyConfig()
	cfg.SetDuration(config.PUBLIC_API_SEND_TRANSACTION_TIMEOUT, 1*time.Millisecond)

	papi := NewPublicApi(cfg, txpMock, vmMock, log)
	txpMock.When("AddNewTransaction", mock.Any).Return(&services.AddNewTransactionOutput{}).Times(1)
	_, err := papi.SendTransaction(&services.SendTransactionInput{
		ClientRequest: (&client.SendTransactionRequestBuilder{
			SignedTransaction: builders.Transaction().Builder()}).Build(),
	})

	require.Error(t, err, "Timeout did not occur")

}
