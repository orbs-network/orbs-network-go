package httpserver

import (
	"time"

	"github.com/orbs-network/membuffers/go"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"github.com/valyala/fasthttp"
	"net/http"
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
		Handler:     server.createRouter(),
		ReadTimeout: time.Millisecond,
	}

	go func() {
		reporting.Info("starting http server on address", log.String("address", address))
		server.httpServer.ListenAndServe(address) //TODO error on failed startup
	}()

	return server
}

func (s *fastHttpServer) sendTransactionHandler(ctx *fasthttp.RequestCtx) {
	bytes, e := readFastInput(ctx)
	if bytes == nil {
		writeFastErrorResponseAndLog(s.reporting, ctx, e)
		return
	}

	clientRequest := client.SendTransactionRequestReader(bytes)
	if e := validate(clientRequest); e != nil {
		writeFastErrorResponseAndLog(s.reporting, ctx, e)
		return
	}

	s.reporting.Info("http server received send-transaction", log.Stringable("request", clientRequest))
	result, err := s.publicApi.SendTransaction(&services.SendTransactionInput{ClientRequest: clientRequest})
	if result != nil && result.ClientResponse != nil {
		writeFastMembuffResponse(ctx, result.ClientResponse, translateStatusToHttpCode(result.ClientResponse.RequestStatus()), result.ClientResponse.StringTransactionStatus())
	} else {
		writeFastErrorResponseAndLog(s.reporting, ctx, &httpErr{http.StatusInternalServerError, log.Error(err), err.Error()})
	}
}

func readFastInput(ctx *fasthttp.RequestCtx) ([]byte, *httpErr) {
	if ctx.PostBody() == nil {
		return nil, &httpErr{http.StatusBadRequest, nil, "http request body is empty"}
	}

	bytes := ctx.PostBody()
	return bytes, nil
}

func (s *fastHttpServer) callMethodHandler(ctx *fasthttp.RequestCtx) {
	bytes, e := readFastInput(ctx)
	if e != nil {
		writeFastErrorResponseAndLog(s.reporting, ctx, e)
		return
	}

	clientRequest := client.CallMethodRequestReader(bytes)
	if e := validate(clientRequest); e != nil {
		writeFastErrorResponseAndLog(s.reporting, ctx, e)
		return
	}

	s.reporting.Info("http server received call-method", log.Stringable("request", clientRequest))
	result, err := s.publicApi.CallMethod(&services.CallMethodInput{ClientRequest: clientRequest})
	if result != nil && result.ClientResponse != nil {
		writeFastMembuffResponse(ctx, result.ClientResponse, translateStatusToHttpCode(result.ClientResponse.RequestStatus()), result.ClientResponse.StringCallMethodResult())
	} else {
		writeFastErrorResponseAndLog(s.reporting, ctx, &httpErr{http.StatusInternalServerError, log.Error(err), err.Error()})
	}
}

func (s *fastHttpServer) createRouter() func(ctx *fasthttp.RequestCtx) {
	return func(ctx *fasthttp.RequestCtx) {
		meter := s.reporting.Meter("request-process-time", log.String("url", string(ctx.Path())))
		defer meter.Done()

		switch string(ctx.Path()) {
		case "/api/v1/send-transaction":
			s.sendTransactionHandler(ctx)
		case "/api/v1/call-method":
			s.callMethodHandler(ctx)
		}
	}
}

func (s *fastHttpServer) GracefulShutdown(timeout time.Duration) {
	s.httpServer.Shutdown() //TODO timeout context
}

func writeFastErrorResponseAndLog(reporting log.BasicLogger, ctx *fasthttp.RequestCtx, m *httpErr) {
	if m.logField == nil {
		reporting.Info(m.message)
	} else {
		reporting.Info(m.message, m.logField)
	}
	ctx.Response.Header.Set("Content-Type", "text/plain")
	ctx.SetStatusCode(m.code)
	ctx.SetBodyString(m.message)
}

func writeFastMembuffResponse(ctx *fasthttp.RequestCtx, message membuffers.Message, httpCode int, orbsText string) {
	ctx.Response.Header.Add("Content-Type", "application/vnd.membuffers")
	ctx.SetStatusCode(httpCode)
	ctx.Response.Header.Add("X-ORBS-CODE-NAME", orbsText)
	ctx.Write(message.Raw())
}
