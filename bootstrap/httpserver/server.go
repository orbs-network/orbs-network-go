// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package httpserver

import (
	"context"
	"fmt"
	membuffers "github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/scribe/log"
)

var LogTag = log.String("adapter", "http-HttpServer")

type httpErr struct {
	code     int
	logField *log.Field
	message  string
}

type HttpServer struct {
	supervised.ChanShutdownWaiter
	httpServer *http.Server
	router     *http.ServeMux

	logger         log.Logger
	publicApi      services.PublicApi
	metricRegistry metric.Registry
	config         config.HttpServerConfig

	port int
}

type TcpKeepAliveListener struct {
	*net.TCPListener
}

func (ln TcpKeepAliveListener) Accept() (net.Conn, error) {
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

func NewHttpServer(cfg config.HttpServerConfig, logger log.Logger, metricRegistry metric.Registry) *HttpServer {
	server := &HttpServer{
		logger:             logger.WithTags(LogTag),
		publicApi:          nil,
		metricRegistry:     metricRegistry,
		config:             cfg,
		ChanShutdownWaiter: supervised.NewChanWaiter("NodeHttpServer"),
	}

	if listener, err := server.listen(server.config.HttpAddress()); err != nil {
		panic(fmt.Sprintf("failed to start http HttpServer: %s", err.Error()))
	} else {
		server.port = listener.Addr().(*net.TCPAddr).Port
		server.router = server.createRouter()
		server.httpServer = &http.Server{
			Handler: server.router,
		}

		// We prefer not to use `HttpServer.ListenAndServe` because we want to block until the socket is listening or exit immediately
		go func() {
			err = server.httpServer.Serve(TcpKeepAliveListener{listener.(*net.TCPListener)})
			if err != nil && err != http.ErrServerClosed {
				logger.Error("failed serving http requests", log.Error(err))
			}
		}()
	}

	logger.Info("started http HttpServer", log.String("address", server.config.HttpAddress()))

	return server
}

func (s *HttpServer) Port() int {
	return s.port
}

func (s *HttpServer) Router() *http.ServeMux {
	return s.router
}

func (s *HttpServer) listen(addr string) (net.Listener, error) {
	return net.Listen("tcp", addr)
}

func (s *HttpServer) GracefulShutdown(shutdownContext context.Context) {
	if err := s.httpServer.Shutdown(shutdownContext); err != nil {
		s.logger.Error("failed to stop http HttpServer gracefully", log.Error(err))
	}
	s.Shutdown()

}

func (s *HttpServer) RegisterPublicApi(publicApi services.PublicApi) {
	s.publicApi = publicApi
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

func (s *HttpServer) wrapHandlerWithPublicApiChecker(f http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if s.publicApi == nil {
			w.WriteHeader(http.StatusServiceUnavailable)
		} else {
			f(w, r)
		}
	}
}

func (s *HttpServer) registerHttpHandler(router *http.ServeMux, urlPath string, withCORS bool, handler http.HandlerFunc) {
	if withCORS {
		handler = wrapHandlerWithCORS(handler)
	}

	if strings.HasPrefix(urlPath, "/api") {
		handler = s.wrapHandlerWithPublicApiChecker(handler)
	}

	router.Handle(urlPath, handler)
}

func (s *HttpServer) createRouter() *http.ServeMux {
	router := http.NewServeMux()

	s.registerHttpHandler(router, "/api/v1/send-transaction", true, s.sendTransactionHandler)
	s.registerHttpHandler(router, "/api/v1/send-transaction-async", true, s.sendTransactionAsyncHandler)
	s.registerHttpHandler(router, "/api/v1/run-query", true, s.runQueryHandler)
	s.registerHttpHandler(router, "/api/v1/get-transaction-status", true, s.getTransactionStatusHandler)
	s.registerHttpHandler(router, "/api/v1/get-transaction-receipt-proof", true, s.getTransactionReceiptProofHandler)
	s.registerHttpHandler(router, "/api/v1/get-block", true, s.getBlockHandler)
	s.registerHttpHandler(router, "/status", true, s.getStatus)
	s.registerHttpHandler(router, "/metrics", true, s.getStatus)
	s.registerHttpHandler(router, "/metrics.json", true, s.getStatus)
	s.registerHttpHandler(router, "/metrics.prometheus", true, s.dumpMetricsAsPrometheus)
	s.registerHttpHandler(router, "/robots.txt", false, s.robots)
	s.registerHttpHandler(router, "/debug/logs/filter-on", false, s.filterOn)
	s.registerHttpHandler(router, "/debug/logs/filter-off", false, s.filterOff)

	router.Handle("/", http.HandlerFunc(wrapHandlerWithCORS(s.Index)))

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
	case protocol.REQUEST_STATUS_NOT_FOUND:
		return http.StatusNotFound
	case protocol.REQUEST_STATUS_RESERVED:
		return http.StatusInternalServerError
	}
	return http.StatusNotImplemented
}

func (s *HttpServer) writeMembuffResponse(w http.ResponseWriter, message membuffers.Message, requestResult *client.RequestResult, errorForVerbosity error) {
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

func (s *HttpServer) writeErrorResponseAndLog(w http.ResponseWriter, m *httpErr) {
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
