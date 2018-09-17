package httpserver

import (
	"bytes"
	"fmt"
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
	"github.com/orbs-network/orbs-network-go/test"
)

func Test_HttpServer_ReadInput_EmptyPost(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	papiMock := &services.MockPublicApi{}

	s := NewHttpServer("", logger, papiMock)

	req, _ := http.NewRequest("POST", "1", nil)
	rec := httptest.NewRecorder()
	s.(*server).readInput(req, rec)

	require.Equal(t, http.StatusBadRequest, rec.Code, "empty body should cause bad request error")
}

func Test_HttpServer_ReadInput_ErrorBodyPost(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	papiMock := &services.MockPublicApi{}

	s := NewHttpServer("", logger, papiMock)

	req, _ := http.NewRequest("POST", "1", errReader(0))
	rec := httptest.NewRecorder()
	s.(*server).readInput(req, rec)

	require.Equal(t, http.StatusBadRequest, rec.Code, "empty body should cause bad request error")
}

type errReader int

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.Errorf("test error")
}

func Test_HttpServer_IsValid_BadMembuff(t *testing.T) {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	papiMock := &services.MockPublicApi{}

	s := NewHttpServer("", logger, papiMock)

	m := client.CallMethodRequestReader([]byte("Random Bytes"))

	rec := httptest.NewRecorder()
	s.(*server).isValid(m, rec)

	require.Equal(t, http.StatusBadRequest, rec.Code, "bad input in body should cause bad request error")
}

func Test_HttpServer_TranslateStatusToHttpCode(t *testing.T) {
	tests := []struct {
		name   string
		expect int
		status protocol.RequestStatus
	}{
		{"REQUEST_STATUS_RESERVED", http.StatusInternalServerError, protocol.REQUEST_STATUS_RESERVED},
		{"REQUEST_STATUS_COMPLETED", http.StatusOK, protocol.REQUEST_STATUS_COMPLETED},
		{"REQUEST_STATUS_IN_PROCESS", http.StatusAccepted, protocol.REQUEST_STATUS_IN_PROCESS},
		{"REQUEST_STATUS_NOT_FOUND", http.StatusNotFound, protocol.REQUEST_STATUS_NOT_FOUND},
		{"REQUEST_STATUS_REJECTED", http.StatusBadRequest, protocol.REQUEST_STATUS_REJECTED},
		{"REQUEST_STATUS_CONGESTION", http.StatusServiceUnavailable, protocol.REQUEST_STATUS_CONGESTION},
	}
	for i := range tests {
		cTest := tests[i] // this is so that we can run tests in parallel, see https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721
		t.Run(cTest.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, cTest.expect, translateStatusToHttpCode(cTest.status), fmt.Sprintf("%s was translated to %d", cTest.name, cTest.expect))
		})
	}
}

func Test_HttpServer_writeMembuffResponse(t *testing.T) {
	expectedResponse := (&client.SendTransactionResponseBuilder{
		RequestStatus:      protocol.REQUEST_STATUS_COMPLETED,
		TransactionReceipt: nil,
		TransactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
		BlockHeight:        1,
		BlockTimestamp:     primitives.TimestampNano(time.Now().Nanosecond()),
	}).Build()

	rec := httptest.NewRecorder()
	writeMembuffResponse(rec, expectedResponse, http.StatusOK, "hello")

	require.Equal(t, http.StatusOK, rec.Code, "code value is not equal")
	require.Equal(t, "application/membuffers", rec.Header().Get("Content-Type"), "should have our content type")
	require.Equal(t, "hello", rec.Header().Get("X-ORBS-CODE-NAME"), "should have correct x-orbs")
	responseFromBody := client.SendTransactionResponseReader(rec.Body.Bytes())
	test.RequireCmpEqual(t, expectedResponse, responseFromBody, "body response and pre-done response are not equal")
}

func Test_HttpServer_writeTextResponse(t *testing.T) {
	rec := httptest.NewRecorder()
	writeTextResponse(rec, "hello test", http.StatusAccepted)
	require.Equal(t, http.StatusAccepted, rec.Code, "code value is not equal")
	require.Equal(t, "plain/text", rec.Header().Get("Content-Type"), "should have our content type")
	require.Equal(t, "hello test", rec.Body.String(), "should have text value")
}

func Test_HttpServer_SendTxHandler_Basic(t *testing.T) {
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

func Test_HttpServer_SendTxHandler_Error(t *testing.T) {
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

func Test_HttpServer_CallMethod_Basic(t *testing.T) {
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

func Test_HttpServer_CallMethod_Error(t *testing.T) {
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

