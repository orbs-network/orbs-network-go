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

	readTimeout, _ := time.ParseDuration("500ms") //TODO error on failed parse
	server.httpServer = &fasthttp.Server{
		Handler:     server.createRouter(),
		ReadTimeout: readTimeout,
	}

	go func() {
		reporting.Info("starting http server on address", log.String("address", address))
		server.httpServer.ListenAndServe(address) //TODO error on failed startup
	}()

	return server
}

//TODO extract commonalities between handlers
func (s *fastHttpServer) createRouter() func(ctx *fasthttp.RequestCtx) {
	sendTransactionHandler := func(ctx *fasthttp.RequestCtx, postBody []byte) {
		clientRequest := client.SendTransactionRequestReader(postBody)
		if reportErrorOnInvalidRequest(clientRequest, ctx) {
			return
		}

		result, err := s.publicApi.SendTransaction(&services.SendTransactionInput{ClientRequest: clientRequest})
		writeMessageOrError(result.ClientResponse, err, ctx)
	}

	callMethodHandler := func(ctx *fasthttp.RequestCtx, postBody []byte) {
		clientRequest := client.CallMethodRequestReader(postBody)
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
		defer meter.Done()

		postBody := ctx.PostBody()
		if postBody == nil {
			s.reporting.Info("could not read http request body")
			ctx.SetStatusCode(fasthttp.StatusInternalServerError)
			return
		}

		switch string(ctx.Path()) {
		case "/api/send-transaction":
			sendTransactionHandler(ctx, postBody)
		case "/api/call-method":
			callMethodHandler(ctx, postBody)
		}
	}
}

func (s *fastHttpServer) GracefulShutdown(timeout time.Duration) {
	s.httpServer.Shutdown() //TODO timeout context
}

func reportErrorOnInvalidRequest(m membuffers.Message, ctx *fasthttp.RequestCtx) bool {
	if !m.IsValid() {
		//TODO report error to Reporting
		ctx.SetStatusCode(fasthttp.StatusBadRequest)
		ctx.SetBody([]byte("Input is invalid"))
		return true
	}

	return false
}

func writeMessageOrError(message membuffers.Message, err error, ctx *fasthttp.RequestCtx) {
	//TODO handle errors
	ctx.Response.Header.Set("Content-Type", "application/octet-stream")
	if err != nil {
		ctx.SetBody([]byte(err.Error()))
	} else {
		ctx.SetBody(message.Raw())
	}
}
