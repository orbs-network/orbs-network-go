package httpserver

import (
	"context"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
)

type HttpServer interface {
	GracefulShutdown(timeout time.Duration)
}

type server struct {
	httpServer *http.Server
	reporting  log.BasicLogger
	publicApi  services.PublicApi
}

func NewHttpServer(address string, reporting log.BasicLogger, publicApi services.PublicApi) HttpServer {
	reporting = reporting.For(log.String("adapter", "http-server"))
	server := &server{
		reporting: reporting,
		publicApi: publicApi,
	}

	server.httpServer = &http.Server{
		Addr:    address,
		Handler: server.createRouter(),
	}

	go func() {
		reporting.Info("starting http server on address", log.String("address", address))
		server.httpServer.ListenAndServe() //TODO error on failed startup
	}()

	return server
}

func (s *server) GracefulShutdown(timeout time.Duration) {
	s.httpServer.Shutdown(context.TODO()) //TODO timeout context
}

func (s *server) createRouter() http.Handler {
	router := http.NewServeMux()
	router.Handle("/v1/api/send-transaction", report(s.reporting, http.HandlerFunc(s.sendTransactionHandler)))
	router.Handle("/v1/api/call-method", report(s.reporting, http.HandlerFunc(s.callMethodHandler)))
	return router
}

func report(reporting log.BasicLogger, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meter := reporting.Meter("request-process-time", log.String("url", r.URL.String()))
		defer meter.Done()
		h.ServeHTTP(w, r)
	})
}

func (s *server) sendTransactionHandler(w http.ResponseWriter, r *http.Request) {
	bytes := s.readInput(r, w)
	if bytes == nil {
		return
	}

	clientRequest := client.SendTransactionRequestReader(bytes)
	if !s.isValid(clientRequest, w) {
		return
	}

	s.reporting.Info("http server received send-transaction", log.Stringable("request", clientRequest))
	result, err := s.publicApi.SendTransaction(&services.SendTransactionInput{ClientRequest: clientRequest})
	if result != nil && result.ClientResponse != nil {
		writeMembuffResponse(w, result.ClientResponse, translateStatusToHttpCode(result.ClientResponse.RequestStatus()), result.ClientResponse.StringTransactionStatus())
	} else {
		writeTextResponse(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) callMethodHandler(w http.ResponseWriter, r *http.Request) {
	bytes := s.readInput(r, w)
	if bytes == nil {
		return
	}

	clientRequest := client.CallMethodRequestReader(bytes)
	if !s.isValid(clientRequest, w) {
		return
	}

	s.reporting.Info("http server received call-method", log.Stringable("request", clientRequest))
	result, err := s.publicApi.CallMethod(&services.CallMethodInput{ClientRequest: clientRequest})
	if result != nil && result.ClientResponse != nil {
		writeMembuffResponse(w, result.ClientResponse, translateStatusToHttpCode(result.ClientResponse.RequestStatus()), result.ClientResponse.StringCallMethodResult())
	} else {
		writeTextResponse(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) readInput(r *http.Request, w http.ResponseWriter) []byte {
	if r.Body == nil {
		s.reporting.Info("could not read empty http request body")
		writeTextResponse(w, "http request body is empty", http.StatusBadRequest)
		return nil
	}

	bytes, err := ioutil.ReadAll(r.Body)
	if err != nil {
		s.reporting.Info("could not read http request body", log.Error(err))
		writeTextResponse(w, err.Error(), http.StatusBadRequest)
		return nil
	}
	return bytes
}

func (s *server) isValid(m membuffers.Message, w http.ResponseWriter) bool {
	if !m.IsValid() {
		s.reporting.Info("http server membuffer not valid", log.Stringable("request", m))
		writeTextResponse(w, "http request body input is invalid", http.StatusBadRequest)
		return false
	}
	return true
}

func translateStatusToHttpCode(responseCode protocol.RequestStatus) int {
	switch responseCode {
	case protocol.REQUEST_STATUS_COMPLETED:
		return http.StatusOK
	case protocol.REQUEST_STATUS_IN_PROCESS:
		return http.StatusAccepted
	case protocol.REQUEST_STATUS_NOT_FOUND:
		return http.StatusNotFound
	case protocol.REQUEST_STATUS_REJECTED:
		return http.StatusBadRequest
	case protocol.REQUEST_STATUS_CONGESTION:
		return http.StatusServiceUnavailable
	case protocol.REQUEST_STATUS_RESERVED:
		return http.StatusInternalServerError
	}
	return http.StatusNotImplemented
}

func writeMembuffResponse(w http.ResponseWriter, message membuffers.Message, httpCode int, orbsText string) {
	w.Header().Set("Content-Type", "application/membuffers")
	w.WriteHeader(httpCode)
	w.Header().Set("X-ORBS-CODE-NAME", orbsText)
	w.Write(message.Raw())
}

func writeTextResponse(w http.ResponseWriter, message string, httpCode int) {
	w.Header().Set("Content-Type", "plain/text")
	w.WriteHeader(httpCode)
	w.Write([]byte(message))
}
