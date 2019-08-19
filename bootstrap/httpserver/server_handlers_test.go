// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package httpserver

import (
	"bytes"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/test/builders"
	"github.com/orbs-network/orbs-network-go/test/with"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHttpServer_SendTransaction_Basic(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		h := newHarness(parent)

		response := &client.SendTransactionResponseBuilder{
			RequestResult:     aCompletedResult(),
			TransactionStatus: protocol.TRANSACTION_STATUS_COMMITTED,
		}
		h.onSendTransaction().Return(&services.SendTransactionOutput{ClientResponse: response.Build()}, nil)

		rec := h.sendTransaction(builders.TransferTransaction().Builder())

		require.Equal(t, http.StatusOK, rec.Code, "should succeed")
	})
}

func TestHttpServer_SendTransaction_Error(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		h := newHarness(parent)

		h.onSendTransaction().Return(nil, errors.Errorf("kaboom"))

		rec := h.sendTransaction(builders.TransferTransaction().Builder())

		require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
	})
}

func TestHttpServer_SendTransactionAsync_Basic(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		h := newHarness(parent)

		response := &client.SendTransactionResponseBuilder{
			RequestResult: &client.RequestResultBuilder{
				RequestStatus:  protocol.REQUEST_STATUS_IN_PROCESS,
				BlockHeight:    1,
				BlockTimestamp: primitives.TimestampNano(time.Now().UnixNano()),
			},
			TransactionStatus: protocol.TRANSACTION_STATUS_PENDING,
		}
		h.onSendTransactionAsync().Return(&services.SendTransactionOutput{ClientResponse: response.Build()})

		rec := h.sendTransactionAsync(builders.TransferTransaction().Builder())

		require.Equal(t, http.StatusAccepted, rec.Code, "should be accepted (202)")
	})
}

func TestHttpServer_RunQuery_Basic(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		h := newHarness(parent)
		response := &client.RunQueryResponseBuilder{
			RequestResult: aCompletedResult(),
			QueryResult: &protocol.QueryResultBuilder{
				ExecutionResult: protocol.EXECUTION_RESULT_SUCCESS,
			},
		}

		h.onRunQuery().Return(&services.RunQueryOutput{ClientResponse: response.Build()})

		rec := h.runQuery()

		require.Equal(t, http.StatusOK, rec.Code, "should succeed")
		// actual values are checked in the server_test.go as unit test of internal WriteMembuffResponse
	})
}

func TestHttpServer_RunQuery_Error(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		h := newHarness(parent)
		h.onRunQuery().Return(nil, errors.Errorf("kaboom"))

		rec := h.runQuery()

		require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
		// actual values are checked in the server_test.go as unit test of internal writeErrorResponseAndLog
	})
}

func TestHttpServer_GetTransactionStatus_Basic(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		h := newHarness(parent)
		response := &client.GetTransactionStatusResponseBuilder{
			RequestResult:     aCompletedResult(),
			TransactionStatus: protocol.TRANSACTION_STATUS_COMMITTED,
		}

		h.onGetTransactionStatus().Return(&services.GetTransactionStatusOutput{ClientResponse: response.Build()})

		rec := h.getTransactionStatus()

		require.Equal(t, http.StatusOK, rec.Code, "should succeed")
		// actual values are checked in the server_test.go as unit test of internal WriteMembuffResponse
	})
}

func TestHttpServer_GetTransactionStatus_Error(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		h := newHarness(parent)
		h.onGetTransactionStatus().Return(nil, errors.Errorf("stam"))

		rec := h.getTransactionStatus()

		require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
		// actual values are checked in the server_test.go as unit test of internal writeErrorResponseAndLog
	})
}

func TestHttpServer_GetTransactionReceiptProof_Basic(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		h := newHarness(parent)

		response := &client.GetTransactionReceiptProofResponseBuilder{
			RequestResult:     aCompletedResult(),
			TransactionStatus: protocol.TRANSACTION_STATUS_COMMITTED,
		}
		h.onGetTransactionReceiptProof().Return(&services.GetTransactionReceiptProofOutput{ClientResponse: response.Build()})

		rec := h.getTransactionReceiptProof()

		require.Equal(t, http.StatusOK, rec.Code, "should succeed")
		// actual values are checked in the server_test.go as unit test of internal WriteMembuffResponse
	})
}

func TestHttpServer_GetTransactionReceiptProof_Error(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		h := newHarness(parent)
		h.onGetTransactionReceiptProof().Return(nil, errors.Errorf("kaboom"))

		rec := h.getTransactionReceiptProof()

		require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
		// actual values are checked in the server_test.go as unit test of internal writeErrorResponseAndLog
	})
}

func TestHttpServer_GetBlock_Basic(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		h := newHarness(parent)
		response := &client.GetBlockResponseBuilder{
			RequestResult: aCompletedResult(),
		}

		h.onGetBlock().Return(&services.GetBlockOutput{ClientResponse: response.Build()})

		rec := h.getBlock()

		require.Equal(t, http.StatusOK, rec.Code, "should succeed")
		// actual values are checked in the server_test.go as unit test of internal WriteMembuffResponse
	})
}

func TestHttpServer_GetBlock_Error(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		h := newHarness(parent)
		h.onGetBlock().Return(nil, errors.Errorf("kaboom"))

		rec := h.getBlock()

		require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
		// actual values are checked in the server_test.go as unit test of internal writeErrorResponseAndLog
	})
}

func TestHttpServer_Index(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		h := newHarness(parent)
		req, _ := http.NewRequest("GET", "/", nil)
		rec := httptest.NewRecorder()
		h.server.Index(rec, req)

		require.Equal(t, http.StatusOK, rec.Code, "should return 200")

		reqNotFound, _ := http.NewRequest("GET", "/does-not-exist", nil)
		recNotFound := httptest.NewRecorder()
		h.server.Index(recNotFound, reqNotFound)

		require.Equal(t, http.StatusNotFound, recNotFound.Code, "should return 404")
	})
}

func TestHttpServer_Robots(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		h := newHarness(parent)

		req, _ := http.NewRequest("Get", "/robots.txt", nil)
		rec := httptest.NewRecorder()
		h.server.robots(rec, req)

		expectedResponse := "User-agent: *\nDisallow: /\n"

		require.Equal(t, http.StatusOK, rec.Code, "should succeed")
		require.Equal(t, "text/plain", rec.Header().Get("Content-Type"), "should have our content type")
		require.Equal(t, expectedResponse, rec.Body.String(), "should have text value")
	})
}

func aCompletedResult() *client.RequestResultBuilder {
	return &client.RequestResultBuilder{
		RequestStatus:  protocol.REQUEST_STATUS_COMPLETED,
		BlockHeight:    1,
		BlockTimestamp: primitives.TimestampNano(time.Now().UnixNano()),
	}
}

type harness struct {
	*with.LoggingHarness
	publicApi *services.MockPublicApi
	server    *HttpServer
}

func (h *harness) onSendTransaction() *mock.MockFunction {
	return h.publicApi.When("SendTransaction", mock.Any, mock.Any).Times(1)
}

func (h *harness) onSendTransactionAsync() *mock.MockFunction {
	return h.publicApi.When("SendTransactionAsync", mock.Any, mock.Any).Times(1)
}

func (h *harness) onGetTransactionStatus() *mock.MockFunction {
	return h.publicApi.When("GetTransactionStatus", mock.Any, mock.Any).Times(1)
}

func (h *harness) onGetBlock() *mock.MockFunction {
	return h.publicApi.When("GetBlock", mock.Any, mock.Any).Times(1)
}

func (h *harness) onGetTransactionReceiptProof() *mock.MockFunction {
	return h.publicApi.When("GetTransactionReceiptProof", mock.Any, mock.Any).Times(1)
}

func (h *harness) onRunQuery() *mock.MockFunction {
	return h.publicApi.When("RunQuery", mock.Any, mock.Any).Times(1)
}

func (h *harness) sendTransaction(builder *protocol.SignedTransactionBuilder) *httptest.ResponseRecorder {
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: builders.TransferTransaction().Builder(),
	}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	h.server.sendTransactionHandler(rec, req)
	return rec
}

func (h *harness) sendTransactionAsync(builder *protocol.SignedTransactionBuilder) *httptest.ResponseRecorder {
	request := (&client.SendTransactionRequestBuilder{
		SignedTransaction: builders.TransferTransaction().Builder(),
	}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	h.server.sendTransactionAsyncHandler(rec, req)
	return rec
}

func (h *harness) runQuery() *httptest.ResponseRecorder {
	request := (&client.RunQueryRequestBuilder{
		SignedQuery: &protocol.SignedQueryBuilder{},
	}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	h.server.runQueryHandler(rec, req)
	return rec
}

func (h *harness) getTransactionStatus() *httptest.ResponseRecorder {
	request := (&client.GetTransactionStatusRequestBuilder{}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	h.server.getTransactionStatusHandler(rec, req)

	return rec
}

func (h *harness) getTransactionReceiptProof() *httptest.ResponseRecorder {
	request := (&client.GetTransactionReceiptProofRequestBuilder{}).Build()

	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	h.server.getTransactionReceiptProofHandler(rec, req)
	return rec
}

func (h *harness) getBlock() *httptest.ResponseRecorder {
	request := (&client.GetBlockRequestBuilder{BlockHeight: 1}).Build()
	req, _ := http.NewRequest("POST", "", bytes.NewReader(request.Raw()))
	rec := httptest.NewRecorder()
	h.server.getBlockHandler(rec, req)
	return rec
}

func newHarness(parent *with.LoggingHarness) *harness {
	papiMock := &services.MockPublicApi{}
	return &harness{
		LoggingHarness: parent,
		publicApi:      papiMock,
		server:         NewHttpServer(NewServerConfig(":0", false), parent.Logger, papiMock, metric.NewRegistry()),
	}
}
