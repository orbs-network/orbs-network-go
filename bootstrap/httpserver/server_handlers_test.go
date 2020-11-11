// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package httpserver

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/config"
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
		withServerHarness(parent, func(h *harness) {
			response := &client.SendTransactionResponseBuilder{
				RequestResult:     aCompletedResult(),
				TransactionStatus: protocol.TRANSACTION_STATUS_COMMITTED,
			}
			h.onSendTransaction().Return(&services.SendTransactionOutput{ClientResponse: response.Build()}, nil)

			rec := h.sendTransaction(builders.TransferTransaction().Builder())

			require.Equal(t, http.StatusOK, rec.Code, "should succeed")
		})
	})
}

func TestHttpServer_SendTransaction_Error(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		withServerHarness(parent, func(h *harness) {
			h.onSendTransaction().Return(nil, errors.Errorf("kaboom"))

			rec := h.sendTransaction(builders.TransferTransaction().Builder())

			require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
		})
	})
}

func TestHttpServer_SendTransactionAsync_Basic(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		withServerHarness(parent, func(h *harness) {
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
	})
}

func TestHttpServer_RunQuery_Basic(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		withServerHarness(parent, func(h *harness) {
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
	})
}

func TestHttpServer_RunQuery_Error(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		withServerHarness(parent, func(h *harness) {
			h.onRunQuery().Return(nil, errors.Errorf("kaboom"))

			rec := h.runQuery()

			require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
			// actual values are checked in the server_test.go as unit test of internal writeErrorResponseAndLog
		})
	})
}

func TestHttpServer_GetTransactionStatus_Basic(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		withServerHarness(parent, func(h *harness) {
			response := &client.GetTransactionStatusResponseBuilder{
				RequestResult:     aCompletedResult(),
				TransactionStatus: protocol.TRANSACTION_STATUS_COMMITTED,
			}

			h.onGetTransactionStatus().Return(&services.GetTransactionStatusOutput{ClientResponse: response.Build()})

			rec := h.getTransactionStatus()

			require.Equal(t, http.StatusOK, rec.Code, "should succeed")
			// actual values are checked in the server_test.go as unit test of internal WriteMembuffResponse
		})
	})
}

func TestHttpServer_GetTransactionStatus_Error(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		withServerHarness(parent, func(h *harness) {
			h.onGetTransactionStatus().Return(nil, errors.Errorf("stam"))

			rec := h.getTransactionStatus()

			require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
			// actual values are checked in the server_test.go as unit test of internal writeErrorResponseAndLog
		})
	})
}

func TestHttpServer_GetTransactionReceiptProof_Basic(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		withServerHarness(parent, func(h *harness) {
			response := &client.GetTransactionReceiptProofResponseBuilder{
				RequestResult:     aCompletedResult(),
				TransactionStatus: protocol.TRANSACTION_STATUS_COMMITTED,
			}
			h.onGetTransactionReceiptProof().Return(&services.GetTransactionReceiptProofOutput{ClientResponse: response.Build()})

			rec := h.getTransactionReceiptProof()

			require.Equal(t, http.StatusOK, rec.Code, "should succeed")
			// actual values are checked in the server_test.go as unit test of internal WriteMembuffResponse
		})
	})
}

func TestHttpServer_GetTransactionReceiptProof_Error(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		withServerHarness(parent, func(h *harness) {
			h.onGetTransactionReceiptProof().Return(nil, errors.Errorf("kaboom"))

			rec := h.getTransactionReceiptProof()

			require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
			// actual values are checked in the server_test.go as unit test of internal writeErrorResponseAndLog
		})
	})
}

func TestHttpServer_GetBlock_Basic(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		withServerHarness(parent, func(h *harness) {
			response := &client.GetBlockResponseBuilder{
				RequestResult: aCompletedResult(),
			}

			h.onGetBlock().Return(&services.GetBlockOutput{ClientResponse: response.Build()})

			rec := h.getBlock()

			require.Equal(t, http.StatusOK, rec.Code, "should succeed")
			// actual values are checked in the server_test.go as unit test of internal WriteMembuffResponse
		})
	})
}

func TestHttpServer_GetBlock_Error(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		withServerHarness(parent, func(h *harness) {
			h.onGetBlock().Return(nil, errors.Errorf("kaboom"))

			rec := h.getBlock()

			require.Equal(t, http.StatusInternalServerError, rec.Code, "should fail with 500")
			// actual values are checked in the server_test.go as unit test of internal writeErrorResponseAndLog
		})
	})
}

func TestHttpServer_Index(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		withServerHarness(parent, func(h *harness) {
			req, _ := http.NewRequest("GET", "/", nil)
			rec := httptest.NewRecorder()
			h.server.Index(rec, req)

			require.Equal(t, http.StatusOK, rec.Code, "should return 200")

			reqNotFound, _ := http.NewRequest("GET", "/does-not-exist", nil)
			recNotFound := httptest.NewRecorder()
			h.server.Index(recNotFound, reqNotFound)

			require.Equal(t, http.StatusNotFound, recNotFound.Code, "should return 404")
		})
	})
}

func TestHttpServer_Robots(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		withServerHarness(parent, func(h *harness) {
			req, _ := http.NewRequest("Get", "/robots.txt", nil)
			rec := httptest.NewRecorder()
			h.server.robots(rec, req)

			expectedResponse := "User-agent: *\nDisallow: /\n"

			require.Equal(t, http.StatusOK, rec.Code, "should succeed")
			require.Equal(t, "text/plain", rec.Header().Get("Content-Type"), "should have our content type")
			require.Equal(t, expectedResponse, rec.Body.String(), "should have text value")
		})
	})
}

func TestHttpServer_PublicApiResponds503UntilRegistered(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		withUnregisteredPublicApiServerHarness(parent, func(h *harness) {
			resp, err := h.GetBlockThroughHTTP()
			require.NoError(t, err, "expected HTTP request to succeed")
			defer resp.Body.Close()

			require.Equal(t, resp.StatusCode, 503, "expected publicapi endpoint to respond with HTTP 503")

			response := &client.GetBlockResponseBuilder{
				RequestResult: aCompletedResult(),
			}
			h.onGetBlock().Return(&services.GetBlockOutput{ClientResponse: response.Build()})

			h.server.RegisterPublicApi(h.publicApi)

			resp, err = h.GetBlockThroughHTTP()
			require.NoError(t, err, "expected HTTP request to succeed")
			defer resp.Body.Close()

			require.Equal(t, resp.StatusCode, 200, "expected publicapi endpoint to respond with http 200")
		})
	})
}

func TestHttpServer_NonPublicApiIsAvailableImmediately(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		withUnregisteredPublicApiServerHarness(parent, func(h *harness) {
			resp, err := h.GetRobotsTxtThroughHTTP()
			require.NoError(t, err, "expected HTTP request to succeed")
			defer resp.Body.Close()

			require.Equal(t, resp.StatusCode, 200, "expected robots.txt endpoint to respond with HTTP 200")
		})
	})
}

func TestHttpServer_PublicApiGetStatus(t *testing.T) {
	with.Logging(t, func(parent *with.LoggingHarness) {
		withServerHarness(parent, func(h *harness) {
			h.AllowErrorsMatching("vc status issue")
			h.server.metricRegistry.NewGauge("test.string.not.real").Update(100)

			// empty
			req, _ := http.NewRequest("Get", "/status", nil)
			rec := httptest.NewRecorder()
			h.server.getStatus(rec, req)
			require.Equal(t, http.StatusOK, rec.Code, "should succeed")
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"), "should have our content type")
			res := make(map[string]interface{})
			json.Unmarshal(rec.Body.Bytes(), &res)
			require.Contains(t, res, "Timestamp")
			require.NotContains(t, res, "Error")

			// no genesis still no error
			now := time.Now()
			bsg := h.server.metricRegistry.NewGaugeWithValue("BlockStorage.FileSystemIndex.LastUpdateTime",
				now.Add(-h.server.config.TransactionPoolTimeBetweenEmptyBlocks()*20).Unix())
			rec = httptest.NewRecorder()
			h.server.getStatus(rec, req)
			require.Equal(t, http.StatusOK, rec.Code, "should succeed")
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"), "should have our content type")
			res = make(map[string]interface{})
			json.Unmarshal(rec.Body.Bytes(), &res)
			require.NotContains(t, res, "Error")
			require.Contains(t, res["Status"], "OK")
			require.NotEmpty(t, res["Payload"])

			// empty genesis means check block storage
			genesis := h.server.metricRegistry.NewGauge("Management.Data.GenesisRefTime")
			rec = httptest.NewRecorder()
			h.server.getStatus(rec, req)
			require.Equal(t, http.StatusOK, rec.Code, "should succeed")
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"), "should have our content type")
			res = make(map[string]interface{})
			json.Unmarshal(rec.Body.Bytes(), &res)
			require.Contains(t, res, "Error")
			require.Contains(t, res["Status"], "Last successful blockstorage update")
			require.NotEmpty(t, res["Payload"])

			// genesis in future - no check
			genesis.Update(1000)
			rec = httptest.NewRecorder()
			h.server.getStatus(rec, req)
			require.Equal(t, http.StatusOK, rec.Code, "should succeed")
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"), "should have our content type")
			res = make(map[string]interface{})
			json.Unmarshal(rec.Body.Bytes(), &res)
			require.NotContains(t, res, "Error")
			require.Contains(t, res["Status"], "OK")
			require.NotEmpty(t, res["Payload"])

			// genesis after current check block storage
			h.server.metricRegistry.NewGauge("Management.Data.CurrentRefTime").Update(2000)
			rec = httptest.NewRecorder()
			h.server.getStatus(rec, req)
			require.Equal(t, http.StatusOK, rec.Code, "should succeed")
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"), "should have our content type")
			res = make(map[string]interface{})
			json.Unmarshal(rec.Body.Bytes(), &res)
			require.Contains(t, res, "Error")
			require.Contains(t, res["Status"], "Last successful blockstorage update")
			require.NotEmpty(t, res["Payload"])

			// all good
			bsg.Update(now.Unix())
			rec = httptest.NewRecorder()
			h.server.getStatus(rec, req)
			require.Equal(t, http.StatusOK, rec.Code, "should succeed")
			require.Equal(t, "application/json", rec.Header().Get("Content-Type"), "should have our content type")
			res = make(map[string]interface{})
			json.Unmarshal(rec.Body.Bytes(), &res)
			require.Contains(t, res, "Timestamp")
			require.NotContains(t, res, "Error")
		})
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

func (h *harness) shutdown() {
	h.server.Shutdown()
}

func (h *harness) buildUrl(urlPath string) string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", h.server.port, urlPath)
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

func (h *harness) GetBlockThroughHTTP() (*http.Response, error) {
	request := (&client.GetBlockRequestBuilder{BlockHeight: 1}).Build()
	httpReq, _ := http.NewRequest("POST", h.buildUrl("/api/v1/get-block"), bytes.NewReader(request.Raw()))
	return http.DefaultClient.Do(httpReq)
}

func (h *harness) GetRobotsTxtThroughHTTP() (*http.Response, error) {
	return http.Get(h.buildUrl("/robots.txt"))
}

func withUnregisteredPublicApiServerHarness(parent *with.LoggingHarness, f func(h *harness)) {
	papiMock := &services.MockPublicApi{}
	h := &harness{
		LoggingHarness: parent,
		publicApi:      papiMock,
		server:         NewHttpServer(generateConfig(), parent.Logger, metric.NewRegistry()),
	}
	defer h.shutdown()
	f(h)
}

func generateConfig() config.OverridableConfig {
	return config.ForGamma(nil, nil, nil, ":0", false, "")
}

func withServerHarness(parent *with.LoggingHarness, f func(h *harness)) {
	withUnregisteredPublicApiServerHarness(parent, func(h *harness) {
		h.server.RegisterPublicApi(h.publicApi)
		f(h)
	})
}
