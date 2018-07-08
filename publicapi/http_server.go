package publicapi

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/orbs-network/orbs-network-go/instrumentation"
	"github.com/orbs-network/orbs-network-go/types"
)

type HttpServer interface {
	GracefulShutdown(timeout time.Duration)
}

type httpServer struct {
	httpServer *http.Server
}

func NewHttpServer(address string, logger instrumentation.Reporting, publicApi PublicApi) HttpServer {

	server := &httpServer{
		httpServer: &http.Server{
			Addr:    address,
			Handler: createRouter(publicApi),
		},
	}

	go func() {
		server.httpServer.ListenAndServe() //TODO error on failed startup
	}()

	logger.Info(fmt.Sprintf("server started on address %s", address))

	return server

}

func createRouter(publicApi PublicApi) http.Handler {
	sendTransactionHandler := func(w http.ResponseWriter, r *http.Request) {
		amountParam := r.URL.Query()["amount"][0]
		amount, _ := strconv.ParseInt(amountParam, 10, 32)
		fmt.Println("SendTransaction maybe?")
		publicApi.SendTransaction(&types.Transaction{
			Value: int(amount),
		})
	}

	callMethodHandler := func(w http.ResponseWriter, r *http.Request) {
		amount := publicApi.CallMethod()
		w.Write([]byte(fmt.Sprintf("%v", amount)))
	}

	router := http.NewServeMux()
	router.HandleFunc("/api/send_transaction", sendTransactionHandler)
	router.HandleFunc("/api/call_method", callMethodHandler)
	return router
}

func (s *httpServer) GracefulShutdown(timeout time.Duration) {
	s.httpServer.Shutdown(context.TODO()) //TODO timeout context
}
