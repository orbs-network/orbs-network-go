// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package httpserver

import (
	"context"
	"fmt"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

var LogTag = log.String("adapter", "http-server")

type httpErr struct {
	code     int
	logField *log.Field
	message  string
}

type HttpServer interface {
	GracefulShutdown(timeout time.Duration)
	Port() int
}

type server struct {
	httpServer     *http.Server
	logger         log.BasicLogger
	publicApi      services.PublicApi
	metricRegistry metric.Registry
	config         config.HttpServerConfig

	port int
}

type tcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln tcpKeepAliveListener) Accept() (net.Conn, error) {
	tc, err := ln.AcceptTCP()
	if err != nil {
		return nil, err
	}
	err = tc.SetKeepAlive(true)
	if err != nil {
		return nil, err
	}
	err = tc.SetKeepAlivePeriod(35 * time.Second)
	if err != nil {
		return nil, err
	}
	return tc, nil
}

func NewHttpServer(cfg config.HttpServerConfig, logger log.BasicLogger, publicApi services.PublicApi, metricRegistry metric.Registry) HttpServer {
	server := &server{
		logger:         logger.WithTags(LogTag),
		publicApi:      publicApi,
		metricRegistry: metricRegistry,
		config:         cfg,
	}

	if listener, err := server.listen(server.config.HttpAddress()); err != nil {
		panic(fmt.Sprintf("failed to start http server: %s", err.Error()))
	} else {
		server.port = listener.Addr().(*net.TCPAddr).Port
		server.httpServer = &http.Server{
			Handler: server.createRouter(),
		}

		// We prefer not to use `HttpServer.ListenAndServe` because we want to block until the socket is listening or exit immediately
		go server.httpServer.Serve(tcpKeepAliveListener{listener.(*net.TCPListener)})
	}

	logger.Info("started http server", log.String("address", server.config.HttpAddress()))

	return server
}

func (s *server) Port() int {
	return s.port
}

func (s *server) listen(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}

func (s *server) GracefulShutdown(timeout time.Duration) {
	ctx := context.Background()
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	if err := s.httpServer.Shutdown(ctx); err != nil {
		s.logger.Error("failed to stop http server gracefully", log.Error(err))
	}
}

func (s *server) createRouter() http.Handler {
	router := http.NewServeMux()
	router.Handle("/api/v1/send-transaction", http.HandlerFunc(wrapHandlerWithCORS(s.sendTransactionHandler)))
	router.Handle("/api/v1/send-transaction-async", http.HandlerFunc(wrapHandlerWithCORS(s.sendTransactionAsyncHandler)))
	router.Handle("/api/v1/run-query", http.HandlerFunc(wrapHandlerWithCORS(s.runQueryHandler)))
	router.Handle("/api/v1/get-transaction-status", http.HandlerFunc(wrapHandlerWithCORS(s.getTransactionStatusHandler)))
	router.Handle("/api/v1/get-transaction-receipt-proof", http.HandlerFunc(wrapHandlerWithCORS(s.getTransactionReceiptProofHandler)))
	router.Handle("/api/v1/get-block", http.HandlerFunc(wrapHandlerWithCORS(s.getBlockHandler)))
	router.Handle("/metrics", http.HandlerFunc(wrapHandlerWithCORS(s.dumpMetrics)))
	router.Handle("/robots.txt", http.HandlerFunc(s.robots))
	router.Handle("/debug/logs/filter-on", http.HandlerFunc(s.filterOn))
	router.Handle("/debug/logs/filter-off", http.HandlerFunc(s.filterOff))

	if s.config.Profiling() {
		registerPprof(router)
	}

	return router
}

func readInput(r *http.Request) ([]byte, *httpErr) {
	if r.Body == nil {
		return nil, &httpErr{http.StatusBadRequest, nil, "http request body is empty"}
	}

	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, &httpErr{http.StatusBadRequest, log.Error(err), "http request body is empty"}
	}
	return bytes, nil
}

func validate(m membuffers.Message) *httpErr {
	if !m.IsValid() {
		return &httpErr{http.StatusBadRequest, log.Stringable("request", m), "http request is not a valid membuffer"}
	}
	return nil
}

func translateRequestStatusToHttpCode(responseCode protocol.RequestStatus) int {
	switch responseCode {
	case protocol.REQUEST_STATUS_COMPLETED:
		return http.StatusOK
	case protocol.REQUEST_STATUS_IN_PROCESS:
		return http.StatusAccepted
	case protocol.REQUEST_STATUS_BAD_REQUEST:
		return http.StatusBadRequest
	case protocol.REQUEST_STATUS_CONGESTION:
		return http.StatusServiceUnavailable
	case protocol.REQUEST_STATUS_SYSTEM_ERROR:
		return http.StatusInternalServerError
	case protocol.REQUEST_STATUS_OUT_OF_SYNC:
		return http.StatusServiceUnavailable
	case protocol.REQUEST_STATUS_RESERVED:
		return http.StatusInternalServerError
	}
	return http.StatusNotImplemented
}

func (s *server) writeMembuffResponse(w http.ResponseWriter, message membuffers.Message, requestResult *client.RequestResult, errorForVerbosity error) {
	httpCode := translateRequestStatusToHttpCode(requestResult.RequestStatus())
	w.Header().Set("Content-Type", "application/membuffers")
	w.Header().Set("X-ORBS-REQUEST-RESULT", requestResult.RequestStatus().String())
	w.Header().Set("X-ORBS-BLOCK-HEIGHT", fmt.Sprintf("%d", requestResult.BlockHeight()))
	w.Header().Set("X-ORBS-BLOCK-TIMESTAMP", sprintfTimestamp(requestResult.BlockTimestamp()))

	if errorForVerbosity != nil {
		w.Header().Set("X-ORBS-ERROR-DETAILS", errorForVerbosity.Error())
	}
	w.WriteHeader(httpCode)
	_, err := w.Write(message.Raw())
	if err != nil {
		s.logger.Info("error writing response", log.Error(err))
	}
}

func sprintfTimestamp(timestamp primitives.TimestampNano) string {
	return time.Unix(0, int64(timestamp)).UTC().Format(time.RFC3339Nano)
}

func (s *server) writeErrorResponseAndLog(w http.ResponseWriter, m *httpErr) {
	if m.logField == nil {
		s.logger.Info(m.message)
	} else {
		s.logger.Info(m.message, m.logField)
	}
	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(m.code)
	_, err := w.Write([]byte(m.message))
	if err != nil {
		s.logger.Info("error writing response", log.Error(err))
	}
}

func registerPprof(router *http.ServeMux) {
	router.HandleFunc("/debug/pprof/", pprof.Index)
	router.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	router.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	router.HandleFunc("/debug/pprof/trace", pprof.Trace)
}

// Allows handler to be called via XHR requests from any host
func wrapHandlerWithCORS(f func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
		} else {
			f(w, r)
		}
	}
}
