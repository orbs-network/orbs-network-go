package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"io/ioutil"
	"net"
	"net/http"
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
	port           int
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

func NewHttpServer(address string, logger log.BasicLogger, publicApi services.PublicApi, metricRegistry metric.Registry) HttpServer {
	server := &server{
		logger:         logger.WithTags(LogTag),
		publicApi:      publicApi,
		metricRegistry: metricRegistry,
	}

	if listener, err := server.listen(address); err != nil {
		logger.Error("failed to start http server", log.Error(err))
		panic(fmt.Sprintf("failed to start http server: %s", err.Error()))
	} else {
		server.port = listener.Addr().(*net.TCPAddr).Port
		server.httpServer = &http.Server{
			Handler: server.createRouter(),
		}

		// We prefer not to use `HttpServer.ListenAndServe` because we want to block until the socket is listening or exit immediately
		go server.httpServer.Serve(tcpKeepAliveListener{listener.(*net.TCPListener)})
	}

	logger.Info("started http server", log.String("address", address))

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
	router.Handle("/api/v1/send-transaction", http.HandlerFunc(s.sendTransactionHandler))
	router.Handle("/api/v1/run-query", http.HandlerFunc(s.runQueryHandler))
	router.Handle("/api/v1/get-transaction-status", http.HandlerFunc(s.getTransactionStatusHandler))
	router.Handle("/api/v1/get-transaction-receipt-proof", http.HandlerFunc(s.getTransactionReceiptProofHandler))
	router.Handle("/metrics", http.HandlerFunc(s.dumpMetrics))
	router.Handle("/robots.txt", http.HandlerFunc(s.robots))
	return router
}

func (s *server) robots(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	_, err := w.Write([]byte("User-agent: *\nDisallow: /\n"))
	if err != nil {
		s.logger.Info("error writing robots.txt response", log.Error(err))
	}
}

func (s *server) dumpMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	bytes, _ := json.Marshal(s.metricRegistry.ExportAll())
	_, err := w.Write(bytes)
	if err != nil {
		s.logger.Info("error writing response", log.Error(err))
	}
}

func (s *server) sendTransactionHandler(w http.ResponseWriter, r *http.Request) {
	bytes, e := readInput(r)
	if e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	clientRequest := client.SendTransactionRequestReader(bytes)
	if e := validate(clientRequest); e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	s.logger.Info("http server received send-transaction", log.Stringable("request", clientRequest))
	result, err := s.publicApi.SendTransaction(r.Context(), &services.SendTransactionInput{ClientRequest: clientRequest})
	if result != nil && result.ClientResponse != nil {
		s.writeMembuffResponse(w, result.ClientResponse, result.ClientResponse.RequestResult(), err)
	} else {
		s.writeErrorResponseAndLog(w, &httpErr{http.StatusInternalServerError, log.Error(err), err.Error()})
	}
}

func (s *server) runQueryHandler(w http.ResponseWriter, r *http.Request) {
	bytes, e := readInput(r)
	if e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	clientRequest := client.RunQueryRequestReader(bytes)
	if e := validate(clientRequest); e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	s.logger.Info("http server received run-query", log.Stringable("request", clientRequest))
	result, err := s.publicApi.RunQuery(r.Context(), &services.RunQueryInput{ClientRequest: clientRequest})
	if result != nil && result.ClientResponse != nil {
		s.writeMembuffResponse(w, result.ClientResponse, result.ClientResponse.RequestResult(), err)
	} else {
		s.writeErrorResponseAndLog(w, &httpErr{http.StatusInternalServerError, log.Error(err), err.Error()})
	}
}

func (s *server) getTransactionStatusHandler(w http.ResponseWriter, r *http.Request) {
	bytes, e := readInput(r)
	if e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	clientRequest := client.GetTransactionStatusRequestReader(bytes)
	if e := validate(clientRequest); e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	s.logger.Info("http server received get-transaction-status", log.Stringable("request", clientRequest))
	result, err := s.publicApi.GetTransactionStatus(r.Context(), &services.GetTransactionStatusInput{ClientRequest: clientRequest})
	if result != nil && result.ClientResponse != nil {
		s.writeMembuffResponse(w, result.ClientResponse, result.ClientResponse.RequestResult(), err)
	} else {
		s.writeErrorResponseAndLog(w, &httpErr{http.StatusInternalServerError, log.Error(err), err.Error()})
	}
}

func (s *server) getTransactionReceiptProofHandler(w http.ResponseWriter, r *http.Request) {
	bytes, e := readInput(r)
	if e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	clientRequest := client.GetTransactionReceiptProofRequestReader(bytes)
	if e := validate(clientRequest); e != nil {
		s.writeErrorResponseAndLog(w, e)
		return
	}

	s.logger.Info("http server received get-transaction-receipt-proof", log.Stringable("request", clientRequest))
	result, err := s.publicApi.GetTransactionReceiptProof(r.Context(), &services.GetTransactionReceiptProofInput{ClientRequest: clientRequest})
	if result != nil && result.ClientResponse != nil {
		s.writeMembuffResponse(w, result.ClientResponse, result.ClientResponse.RequestResult(), err)
	} else {
		s.writeErrorResponseAndLog(w, &httpErr{http.StatusInternalServerError, log.Error(err), err.Error()})
	}
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
