package httpserver

import (
	"io/ioutil"
	"net/http"
	"time"

	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/valyala/fasthttp"
)

type FastHttpServer interface {
	GracefulShutdown(timeout time.Duration)
}

type fastHttpServer struct {
	httpServer *fasthttp.Server
	reporting  log.BasicLogger
	publicApi  services.PublicApi
}

func NewFastHttpServer(address string, reporting log.BasicLogger, publicApi services.PublicApi) FastHttpServer {
	server := &fastHttpServer{
		reporting: reporting.For(log.String("subsystem", "http-server")),
		publicApi: publicApi,
	}

	requestHandler := func(ctx *fasthttp.RequestCtx) {
	}

	server.httpServer = &fasthttp.Server{
		Handler: requestHandler,
	}

	go func() {
		reporting.Info("Starting server on address", log.String("address", address))
		server.httpServer.ListenAndServe(address) //TODO error on failed startup
	}()

	return server
}

//TODO extract commonalities between handlers
func (s *fastHttpServer) createRouter() http.Handler {
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
		if result != nil {
			r.writeMessageOrError(result.ClientResponse, err)
		} else {
			r.writeMessageOrError(nil, err)
		}
	})

	router := http.NewServeMux()
	router.Handle("/api/send-transaction", report(s.reporting, sendTransactionHandler))
	router.Handle("/api/call-method", report(s.reporting, callMethodHandler))
	return router
}

func (s *fastHttpServer) GracefulShutdown(timeout time.Duration) {
	s.httpServer.Shutdown() //TODO timeout context
}

func report(reporting log.BasicLogger, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		meter := reporting.Meter("request-process-time", log.String("url", r.URL.String()))
		defer meter.Done()
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
	if err != nil {
		r.writer.Write([]byte(err.Error()))
	} else {
		r.writer.Write(message.Raw())
	}
}

func (s *fastHttpServer) handler(handler func(bytes []byte, r *response)) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bytes, err := ioutil.ReadAll(r.Body)
		if err != nil {
			s.reporting.Info("could not read http request body", log.Error(err))
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
