package httpserver

import (
	"bytes"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
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

func makeServer(papiMock *services.MockPublicApi) HttpServer {
	logger := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stdout, log.NewHumanReadableFormatter()))

	return NewHttpServer(NewServerConfig(":0", false), logger, papiMock, metric.NewRegistry())
}

func TestHttpServer_Robots(t *testing.T) {
	s := makeServer(nil)

	req, _ := http.NewRequest("Get", "/robots.txt", nil)
	rec := httptest.NewRecorder()
	s.(*server).robots(rec, req)

	expectedResponse := "User-agent: *\nDisallow: /\n"

	require.Equal(t, http.StatusOK, rec.Code, "should succeed")
	require.Equal(t, "text/plain", rec.Header().Get("Content-Type"), "should have our content type")
	require.Equal(t, expectedResponse, rec.Body.String(), "should have text value")
}

func TestHttpServerSendTransaction_Basic(t *testing.T) {
	papiMock := &services.MockPublicApi{}
	response := &client.SendTransactionResponseBuilder{
		RequestResult: &client.RequestResultBuilder{
			RequestStatus:  protocol.REQUEST_STATUS_COMPLETED,
			BlockHeight:    1,
			BlockTimestamp: primitives.TimestampNano(time.Now().Nanosecond()),
		},
		TransactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
		TransactionReceipt: nil,
	}

	papiMock.When("SendTransaction", mock.Any, mock.Any).Times(1).Return(&services.SendTransactionOutput{ClientResponse: response.Build()})

	s := makeServer(papiMock)

	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: builders.TransferTransaction().Builder(),
	}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	s.(*server).sendTransactionHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "should succeed")
}

func TestHttpServerSendTransaction_Error(t *testing.T) {
	papiMock := &services.MockPublicApi{}

	papiMock.When("SendTransaction", mock.Any, mock.Any).Times(1).Return(nil, errors.Errorf("stam"))

	s := makeServer(papiMock)

	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: builders.TransferTransaction().Builder(),
	}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	s.(*server).sendTransactionHandler(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
}

func TestHttpServerRunQuery_Basic(t *testing.T) {
	papiMock := &services.MockPublicApi{}
	response := &client.RunQueryResponseBuilder{
		RequestResult: &client.RequestResultBuilder{
			RequestStatus:  protocol.REQUEST_STATUS_COMPLETED,
			BlockHeight:    1,
			BlockTimestamp: primitives.TimestampNano(time.Now().Nanosecond()),
		},
		QueryResult: &protocol.QueryResultBuilder{
			ExecutionResult:     protocol.EXECUTION_RESULT_SUCCESS,
			OutputArgumentArray: nil,
			OutputEventsArray:   nil,
		},
	}

	papiMock.When("RunQuery", mock.Any, mock.Any).Times(1).Return(&services.RunQueryOutput{ClientResponse: response.Build()})

	s := makeServer(papiMock)

	request := (&client.RunQueryRequestBuilder{
		SignedQuery: &protocol.SignedQueryBuilder{},
	}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	s.(*server).runQueryHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "should succeed")
	// actual values are checked in the server_test.go as unit test of internal WriteMembuffResponse
}

func TestHttpServerRunQuery_Error(t *testing.T) {
	papiMock := &services.MockPublicApi{}

	papiMock.When("RunQuery", mock.Any, mock.Any).Times(1).Return(nil, errors.Errorf("stam"))

	s := makeServer(papiMock)

	request := (&client.RunQueryRequestBuilder{
		SignedQuery: &protocol.SignedQueryBuilder{},
	}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	s.(*server).runQueryHandler(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
	// actual values are checked in the server_test.go as unit test of internal writeErrorResponseAndLog
}

func TestHttpServerGetTransactionStatus_Basic(t *testing.T) {
	papiMock := &services.MockPublicApi{}
	response := &client.GetTransactionStatusResponseBuilder{
		RequestResult: &client.RequestResultBuilder{
			RequestStatus:  protocol.REQUEST_STATUS_COMPLETED,
			BlockHeight:    1,
			BlockTimestamp: primitives.TimestampNano(time.Now().Nanosecond()),
		},
		TransactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
		TransactionReceipt: nil,
	}

	papiMock.When("GetTransactionStatus", mock.Any, mock.Any).Times(1).Return(&services.GetTransactionStatusOutput{ClientResponse: response.Build()})

	s := makeServer(papiMock)

	request := (&client.GetTransactionStatusRequestBuilder{}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	s.(*server).getTransactionStatusHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "should succeed")
	// actual values are checked in the server_test.go as unit test of internal WriteMembuffResponse
}

func TestHttpServerGetTransactionStatus_Error(t *testing.T) {
	papiMock := &services.MockPublicApi{}

	papiMock.When("GetTransactionStatus", mock.Any, mock.Any).Times(1).Return(nil, errors.Errorf("stam"))

	s := makeServer(papiMock)

	request := (&client.GetTransactionStatusRequestBuilder{}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	s.(*server).getTransactionStatusHandler(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
	// actual values are checked in the server_test.go as unit test of internal writeErrorResponseAndLog
}

func TestHttpServerGetTransactionReceiptProof_Basic(t *testing.T) {
	papiMock := &services.MockPublicApi{}
	response := &client.GetTransactionReceiptProofResponseBuilder{
		RequestResult: &client.RequestResultBuilder{
			RequestStatus:  protocol.REQUEST_STATUS_COMPLETED,
			BlockHeight:    1,
			BlockTimestamp: primitives.TimestampNano(time.Now().Nanosecond()),
		},
		TransactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
		TransactionReceipt: nil,
		PackedProof:        nil,
	}

	papiMock.When("GetTransactionReceiptProof", mock.Any, mock.Any).Times(1).Return(&services.GetTransactionReceiptProofOutput{ClientResponse: response.Build()})

	s := makeServer(papiMock)

	request := (&client.GetTransactionReceiptProofRequestBuilder{}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	s.(*server).getTransactionReceiptProofHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "should succeed")
	// actual values are checked in the server_test.go as unit test of internal WriteMembuffResponse
}

func TestHttpServerGetTransactionReceiptProof_Error(t *testing.T) {
	papiMock := &services.MockPublicApi{}

	papiMock.When("GetTransactionReceiptProof", mock.Any, mock.Any).Times(1).Return(nil, errors.Errorf("stam"))

	s := makeServer(papiMock)

	request := (&client.GetTransactionReceiptProofRequestBuilder{}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	s.(*server).getTransactionReceiptProofHandler(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
	// actual values are checked in the server_test.go as unit test of internal writeErrorResponseAndLog
}

func TestHttpServerGetBlock_Basic(t *testing.T) {
	papiMock := &services.MockPublicApi{}
	response := &client.GetBlockResponseBuilder{
		RequestResult: &client.RequestResultBuilder{
			RequestStatus:  protocol.REQUEST_STATUS_COMPLETED,
			BlockHeight:    1,
			BlockTimestamp: primitives.TimestampNano(time.Now().Nanosecond()),
		},
		TransactionsBlockHeader:   nil,
		TransactionsBlockMetadata: nil,
		SignedTransactions:        nil,
		TransactionsBlockProof:    nil,
		ResultsBlockHeader:        nil,
		TransactionReceipts:       nil,
		ContractStateDiffs:        nil,
		ResultsBlockProof:         nil,
	}

	papiMock.When("GetBlock", mock.Any, mock.Any).Times(1).Return(&services.GetBlockOutput{ClientResponse: response.Build()})

	s := makeServer(papiMock)

	request := (&client.GetBlockRequestBuilder{BlockHeight: 1}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	s.(*server).getBlockHandler(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "should succeed")
	// actual values are checked in the server_test.go as unit test of internal WriteMembuffResponse
}

func TestHttpServerGetBlock_Error(t *testing.T) {
	papiMock := &services.MockPublicApi{}

	papiMock.When("GetBlock", mock.Any, mock.Any).Times(1).Return(nil, errors.Errorf("stam"))

	s := makeServer(papiMock)

	request := (&client.GetBlockRequestBuilder{}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	s.(*server).getBlockHandler(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
	// actual values are checked in the server_test.go as unit test of internal writeErrorResponseAndLog
}
