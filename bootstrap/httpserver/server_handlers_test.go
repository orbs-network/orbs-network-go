package httpserver

import (
	"bytes"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestHttpServerSendTxHandler_Basic(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	papiMock := &services.MockPublicApi{}
	response := &client.SendTransactionResponseBuilder{
		RequestStatus:      protocol.REQUEST_STATUS_COMPLETED,
		TransactionReceipt: nil,
		TransactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
		BlockHeight:        1,
		BlockTimestamp:     primitives.TimestampNano(time.Now().Nanosecond()),
	}

	papiMock.When("SendTransaction", mock.Any).Times(1).Return(&services.SendTransactionOutput{ClientResponse: response.Build()})

	s := NewHttpServer("", logger, papiMock)

	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: builders.TransferTransaction().Builder(),
	}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	s.(*server).sendTransactionHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "should succeed")
}

func TestHttpServerSendTxHandler_Error(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	papiMock := &services.MockPublicApi{}

	papiMock.When("SendTransaction", mock.Any).Times(1).Return(nil, errors.Errorf("stam"))

	s := NewHttpServer("", logger, papiMock)

	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: builders.TransferTransaction().Builder(),
	}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	s.(*server).sendTransactionHandler(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
}

func TestHttpServerCallMethod_Basic(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	papiMock := &services.MockPublicApi{}
	response := &client.CallMethodResponseBuilder{
		RequestStatus:       protocol.REQUEST_STATUS_COMPLETED,
		OutputArgumentArray: nil,
		CallMethodResult:    protocol.EXECUTION_RESULT_SUCCESS,
		BlockHeight:         1,
		BlockTimestamp:      primitives.TimestampNano(time.Now().Nanosecond()),
	}

	papiMock.When("CallMethod", mock.Any).Times(1).Return(&services.CallMethodOutput{ClientResponse: response.Build()})

	s := NewHttpServer("", logger, papiMock)

	request := (&client.CallMethodRequestBuilder{
		Transaction: &protocol.TransactionBuilder{},
	}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	s.(*server).callMethodHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "should succeed")
}

func TestHttpServerCallMethod_Error(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	papiMock := &services.MockPublicApi{}

	papiMock.When("CallMethod", mock.Any).Times(1).Return(nil, errors.Errorf("stam"))

	s := NewHttpServer("", logger, papiMock)

	request := (&client.CallMethodRequestBuilder{
		Transaction: &protocol.TransactionBuilder{},
	}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	s.(*server).callMethodHandler(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
}


func TestHttpServerGetTx_Basic(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	papiMock := &services.MockPublicApi{}
	response := &client.GetTransactionStatusResponseBuilder{
		RequestStatus:      protocol.REQUEST_STATUS_COMPLETED,
		TransactionReceipt: nil,
		TransactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
		BlockHeight:        1,
		BlockTimestamp:     primitives.TimestampNano(time.Now().Nanosecond()),
	}

	papiMock.When("GetTransactionStatus", mock.Any).Times(1).Return(&services.GetTransactionStatusOutput{ClientResponse: response.Build()})

	s := NewHttpServer("", logger, papiMock)

	request := (&client.GetTransactionStatusRequestBuilder{}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	s.(*server).getTransactionStatusHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "should succeed")
}

func TestHttpServerGetTx_Error(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	papiMock := &services.MockPublicApi{}

	papiMock.When("GetTransactionStatus", mock.Any).Times(1).Return(nil, errors.Errorf("stam"))

	s := NewHttpServer("", logger, papiMock)

	request := (&client.GetTransactionStatusRequestBuilder{}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	s.(*server).getTransactionStatusHandler(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
}
