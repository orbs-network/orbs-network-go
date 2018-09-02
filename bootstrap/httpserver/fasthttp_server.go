package httpserver

import (
	"time"

	"github.com/buaazp/fasthttprouter"
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

func fastSendTransactionHandler(r *fastResponse) {
	clientRequest := client.SendTransactionRequestReader(r.writer.Request.Body())
	if r.reportErrorOnInvalidRequest(clientRequest) {
		return
	}

	result, err := s.publicApi.SendTransaction(&services.SendTransactionInput{ClientRequest: clientRequest})
	r.writeMessageOrError(result.ClientResponse, err)
}

//TODO extract commonalities between handlers
func (s *fastHttpServer) createRouter() *fasthttprouter.Router {
	sendTransactionHandler := s.handler(func(bytes []byte, r *fastResponse) {

		clientRequest := client.SendTransactionRequestReader(bytes)
		if r.reportErrorOnInvalidRequest(clientRequest) {
			return
		}

		result, err := s.publicApi.SendTransaction(&services.SendTransactionInput{ClientRequest: clientRequest})
		r.writeMessageOrError(result.ClientResponse, err)
	})

	callMethodHandler := s.handler(func(bytes []byte, r *fastResponse) {

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

	router := fasthttprouter.New()
	router.Handle("get", "/api/send-transaction", fastReport(sendTransactionHandler, s.reporting))
	router.Handle("get", "/api/call-method", fastReport(callMethodHandler, s.reporting))
	return router
}

func (s *fastHttpServer) GracefulShutdown(timeout time.Duration) {
	s.httpServer.Shutdown() //TODO timeout context
}

func fastReport(h fasthttp.RequestHandler, reporting log.BasicLogger) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {
		meter := reporting.Meter("request-process-time", log.String("url", ctx.URI().String()))
		defer meter.Done()
		h(ctx)

	})
}

func (r *fastResponse) reportErrorOnInvalidRequest(m membuffers.Message) bool {
	if !m.IsValid() {
		//TODO report error to Reporting
		r.writer.Response.SetStatusCode(fasthttp.StatusBadRequest)
		r.writer.Write([]byte("Input is invalid"))
		return true
	}

	return false
}

func (r *fastResponse) writeMessageOrError(message membuffers.Message, err error) {
	//TODO handle errors
	r.writer.Response.Header.Set("Content-Type", "application/octet-stream")
	if err != nil {
		r.writer.Write([]byte(err.Error()))
	} else {
		r.writer.Write(message.Raw())
	}
}

func (s *fastHttpServer) handler(handler func(bytes []byte, r *fastResponse)) fasthttp.RequestHandler {
	return fasthttp.RequestHandler(func(ctx *fasthttp.RequestCtx) {

		bytes := ctx.Request.Body()
		//if err != nil {
		//	s.reporting.Info("could not read http request body", log.Error(err))
		//	w.WriteHeader(http.StatusInternalServerError)
		//	w.Write([]byte(err.Error()))
		//	return
		//}

		handler(bytes, &fastResponse{writer: *ctx})
	})
}

type fastResponse struct {
	writer fasthttp.RequestCtx
}
