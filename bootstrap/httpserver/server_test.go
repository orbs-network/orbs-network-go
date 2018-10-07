package httpserver

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestHttpServerReadInput_EmptyPost(t *testing.T) {
	req, _ := http.NewRequest("POST", "1", nil)
	_, e := readInput(req)

	require.Equal(t, http.StatusBadRequest, e.code, "empty body should cause bad request error")
}

func TestHttpServerReadInput_ErrorBodyPost(t *testing.T) {
	req, _ := http.NewRequest("POST", "1", errReader(0))
	_, e := readInput(req)

	require.Equal(t, http.StatusBadRequest, e.code, "empty body should cause bad request error")
}

type errReader int

func (errReader) Read(p []byte) (n int, err error) {
	return 0, errors.Errorf("test error")
}

func TestHttpServerIsValid_BadMembuff(t *testing.T) {
	m := client.CallMethodRequestReader([]byte("Random Bytes"))
	e := validate(m)

	require.Equal(t, http.StatusBadRequest, e.code, "bad input in body should cause bad request error")
}

func TestHttpServerTranslateStatusToHttpCode(t *testing.T) {
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
		{"REQUEST_STATUS_SYSTEM_ERROR", http.StatusInternalServerError, protocol.REQUEST_STATUS_SYSTEM_ERROR},
	}
	for i := range tests {
		cTest := tests[i] // this is so that we can run tests in parallel, see https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721
		t.Run(cTest.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, cTest.expect, translateStatusToHttpCode(cTest.status), fmt.Sprintf("%s was translated to %d", cTest.name, cTest.expect))
		})
	}
}

func TestHttpServerWriteMembuffResponse(t *testing.T) {
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
	require.Equal(t, "application/vnd.membuffers", rec.Header().Get("Content-Type"), "should have our content type")
	require.Equal(t, "hello", rec.Header().Get("X-ORBS-CODE-NAME"), "should have correct x-orbs")
	responseFromBody := client.SendTransactionResponseReader(rec.Body.Bytes())
	test.RequireCmpEqual(t, expectedResponse, responseFromBody, "body response and pre-done response are not equal")
}

func TestHttpServerWriteTextResponse(t *testing.T) {
	e := &httpErr{
		code:     http.StatusAccepted,
		logField: nil,
		message:  "hello test",
	}
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	rec := httptest.NewRecorder()
	writeErrorResponseAndLog(logger, rec, e)
	require.Equal(t, http.StatusAccepted, rec.Code, "code value is not equal")
	require.Equal(t, "text/plain", rec.Header().Get("Content-Type"), "should have our content type")
	require.Equal(t, "hello test", rec.Body.String(), "should have text value")
}
