package httpserver

import (
	"context"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/orbs-network/membuffers/go"
)

type HttpServer interface {
	GracefulShutdown(timeout time.Duration)
}

type server struct {
	httpServer *http.Server
	reporting  instrumentation.Reporting
	publicApi  services.PublicApi
}

func NewHttpServer(address string, reporting instrumentation.Reporting, publicApi services.PublicApi) HttpServer {
	server := &server{
		reporting: reporting,
		publicApi: publicApi,
	}

	server.httpServer = &http.Server{
		Addr:    address,
		Handler: server.createRouter(),
	}

	go func() {
		reporting.Infof("starting server on address %s", address)
		server.httpServer.ListenAndServe() //TODO error on failed startup
	}()

	return server
}

//TODO extract commonalities between handlers
func (s *server) createRouter() http.Handler {
	sendTransactionHandler := s.handler(func(bytes []byte, r *response) {

		clientRequest := client.SendTransactionRequestReader(bytes)
		if r.reportErrorOnInvalidRequest(clientRequest) {
			return
		}

		result, err := s.publicApi.SendTransaction(&services.SendTransactionInput{ClientRequest: clientRequest})
		r.writeMessageOrError(result.ClientResponse, err)
	})

	callMethodHandler := s.handler(func(bytes []byte, r *response) {

		clientRequest := client.CallMethodRequestReader(bytes)
		if r.reportErrorOnInvalidRequest(clientRequest) {
			return
		}

		result, err := s.publicApi.CallMethod(&services.CallMethodInput{ClientRequest: clientRequest})
		r.writeMessageOrError(result.ClientResponse, err)
	})

	router := http.NewServeMux()
	router.Handle("/api/send-transaction", report(s.reporting, sendTransactionHandler))
	router.Handle("/api/call-method", report(s.reporting, callMethodHandler))
	return router
}

func (s *server) GracefulShutdown(timeout time.Duration) {
	s.httpServer.Shutdown(context.TODO()) //TODO timeout context
}

func report(reporting instrumentation.Reporting, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		reporting.Infof("before %s", r.URL)
		defer func() {
			reporting.Infof("after %s, took %s", r.URL, time.Since(start))
		}()
		h.ServeHTTP(w, r)
	})
}

func (r *response) reportErrorOnInvalidRequest(m membuffers.Message) bool {
	if !m.IsValid() {
		//TODO report error to Reporting
		r.writer.WriteHeader(http.StatusBadRequest)
		r.writer.Write([]byte("Input is invalid"))
		return true
	}

	return false
}

func (r *response) writeMessageOrError(message membuffers.Message, err error) {
	//TODO handle errors
	r.writer.Header().Set("Content-Type", "application/octet-stream")
	r.writer.Write(message.Raw())
}

func (s *server) handler(handler func(bytes []byte, r *response)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			s.reporting.Error(err)
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(err.Error()))
			return
		}

		handler(bytes, &response{writer: w})
	})
}

type response struct {
	writer http.ResponseWriter
}
