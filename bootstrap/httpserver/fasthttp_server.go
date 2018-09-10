package httpserver

import (
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
		reporting: reporting.For(log.String("adapter", "http-server")),
		publicApi: publicApi,
	}

	server.httpServer = &fasthttp.Server{
		Handler: server.createRouter(),
	}

	go func() {
		reporting.Info("starting http server on address", log.String("address", address))
		server.httpServer.ListenAndServe(address) //TODO error on failed startup
	}()

	return server
}

//TODO extract commonalities between handlers
func (s *fastHttpServer) createRouter() func(ctx *fasthttp.RequestCtx) {
	sendTransactionHandler := func(ctx *fasthttp.RequestCtx) {
		clientRequest := client.SendTransactionRequestReader(ctx.PostBody())
		if reportErrorOnInvalidRequest(clientRequest, ctx) {
			return
		}

		result, err := s.publicApi.SendTransaction(&services.SendTransactionInput{ClientRequest: clientRequest})
		writeMessageOrError(result.ClientResponse, err, ctx)
	}

	callMethodHandler := func(ctx *fasthttp.RequestCtx) {
		clientRequest := client.CallMethodRequestReader(ctx.PostBody())
		if reportErrorOnInvalidRequest(clientRequest, ctx) {
			return
		}

		result, err := s.publicApi.CallMethod(&services.CallMethodInput{ClientRequest: clientRequest})
		if result != nil {
			writeMessageOrError(result.ClientResponse, err, ctx)
		} else {
			writeMessageOrError(nil, err, ctx)
		}
	}

	return func(ctx *fasthttp.RequestCtx) {
		meter := s.reporting.Meter("request-process-time", log.String("url", string(ctx.Path())))
		switch string(ctx.Path()) {
		case "/api/send-transaction":
			sendTransactionHandler(ctx)
		case "/api/call-method":
			callMethodHandler(ctx)
		}
		meter.Done()

	}
}

func (s *fastHttpServer) GracefulShutdown(timeout time.Duration) {
	s.httpServer.Shutdown() //TODO timeout context
}

func reportErrorOnInvalidRequest(m membuffers.Message, ctx *fasthttp.RequestCtx) bool {
	if !m.IsValid() {
		//TODO report error to Reporting
		ctx.Response.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.Write([]byte("Input is invalid"))
		return true
	}

	return false
}

func writeMessageOrError(message membuffers.Message, err error, ctx *fasthttp.RequestCtx) {
	//TODO handle errors
	ctx.Response.Header.Set("Content-Type", "application/octet-stream")
	if err != nil {
		ctx.Write([]byte(err.Error()))
	} else {
		ctx.Write(message.Raw())
	}
}
