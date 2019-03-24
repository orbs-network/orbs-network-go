// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package httpserver

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/test"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
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
	m := client.RunQueryRequestReader([]byte("Random Bytes"))
	e := validate(m)

	require.Equal(t, http.StatusBadRequest, e.code, "bad input in body should cause bad request error")
}

func TestHttpServerMethodOptions(t *testing.T) {
	f := func(w http.ResponseWriter, r *http.Request) {
		t.Log("should not be called")
		t.Fail()
	}

	req, _ := http.NewRequest(http.MethodOptions, "1", errReader(0))
	res := httptest.NewRecorder()
	wrapHandlerWithCORS(f)(res, req)

	require.Equal(t, http.StatusOK, res.Code, "always respond OK for options request")
	require.Equal(t, "*", res.Header().Get("Access-Control-Allow-Origin"), "set CORS header for origin")
	require.Equal(t, "*", res.Header().Get("Access-Control-Allow-Headers"), "set CORS header for headers")
	require.Equal(t, "*", res.Header().Get("Access-Control-Allow-Methods"), "set CORS header for methods")
	require.Empty(t, res.Body, "empty response")
}

func TestHttpServerCORS(t *testing.T) {
	f := func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello"))
	}

	req, _ := http.NewRequest(http.MethodPost, "1", errReader(0))
	res := httptest.NewRecorder()
	wrapHandlerWithCORS(f)(res, req)

	require.Equal(t, http.StatusOK, res.Code, "always respond OK for options request")
	require.Equal(t, "*", res.Header().Get("Access-Control-Allow-Origin"), "set CORS header for origin")
	require.Equal(t, "*", res.Header().Get("Access-Control-Allow-Headers"), "set CORS header for headers")
	require.Equal(t, "*", res.Header().Get("Access-Control-Allow-Methods"), "set CORS header for methods")
	require.Equal(t, "hello", res.Body.String(), "expected response from a wrapped function")
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
		{"REQUEST_STATUS_BAD_REQUEST", http.StatusBadRequest, protocol.REQUEST_STATUS_BAD_REQUEST},
		{"REQUEST_STATUS_CONGESTION", http.StatusServiceUnavailable, protocol.REQUEST_STATUS_CONGESTION},
		{"REQUEST_STATUS_SYSTEM_ERROR", http.StatusInternalServerError, protocol.REQUEST_STATUS_SYSTEM_ERROR},
		{"REQUEST_STATUS_OUT_OF_SYNC", http.StatusServiceUnavailable, protocol.REQUEST_STATUS_OUT_OF_SYNC},
	}
	for i := range tests {
		cTest := tests[i] // this is so that we can run tests in parallel, see https://gist.github.com/posener/92a55c4cd441fc5e5e85f27bca008721
		t.Run(cTest.name, func(t *testing.T) {
			t.Parallel()
			require.Equal(t, cTest.expect, translateRequestStatusToHttpCode(cTest.status), fmt.Sprintf("%s was translated to %d", cTest.name, cTest.expect))
		})
	}
}

func mockServer(tb testing.TB) *server {
	logger := log.DefaultTestingLogger(tb)
	return &server{
		logger: logger.WithTags(LogTag),
	}
}

func TestHttpServerWriteMembuffResponse(t *testing.T) {
	expectedResponse := (&client.SendTransactionResponseBuilder{
		RequestResult: &client.RequestResultBuilder{
			RequestStatus:  protocol.REQUEST_STATUS_COMPLETED,
			BlockHeight:    1234,
			BlockTimestamp: 1546858355859000000,
		},
		TransactionStatus:  protocol.TRANSACTION_STATUS_COMMITTED,
		TransactionReceipt: nil,
	}).Build()

	s := mockServer(t)
	rec := httptest.NewRecorder()
	s.writeMembuffResponse(rec, expectedResponse, expectedResponse.RequestResult(), errors.New("example error"))

	require.Equal(t, http.StatusOK, rec.Code, "code value is not equal")
	require.Equal(t, "application/membuffers", rec.Header().Get("Content-Type"), "should have our content type")
	require.Equal(t, "REQUEST_STATUS_COMPLETED", rec.Header().Get("X-ORBS-REQUEST-RESULT"), "should have correct X-ORBS-REQUEST-RESULT")
	require.Equal(t, "1234", rec.Header().Get("X-ORBS-BLOCK-HEIGHT"), "should have correct X-ORBS-BLOCK-HEIGHT")
	require.Equal(t, "2019-01-07T10:52:35.859Z", rec.Header().Get("X-ORBS-BLOCK-TIMESTAMP"), "should have correct X-ORBS-BLOCK-TIMESTAMP")
	require.Equal(t, "example error", rec.Header().Get("X-ORBS-ERROR-DETAILS"), "should have correct X-ORBS-ERROR-DETAILS")
	responseFromBody := client.SendTransactionResponseReader(rec.Body.Bytes())
	test.RequireCmpEqual(t, expectedResponse, responseFromBody, "body response and pre-done response are not equal")
}

func TestHttpServerWriteTextResponse(t *testing.T) {
	e := &httpErr{
		code:     http.StatusAccepted,
		logField: nil,
		message:  "hello test",
	}
	s := mockServer(t)
	rec := httptest.NewRecorder()
	s.writeErrorResponseAndLog(rec, e)
	require.Equal(t, http.StatusAccepted, rec.Code, "code value is not equal")
	require.Equal(t, "text/plain", rec.Header().Get("Content-Type"), "should have our content type")
	require.Equal(t, "hello test", rec.Body.String(), "should have text value")
}
