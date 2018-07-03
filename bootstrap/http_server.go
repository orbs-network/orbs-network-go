package bootstrap

import (
	"net/http"
	"github.com/orbs-network/orbs-network-go/publicapi"
	"github.com/orbs-network/orbs-network-go/types"
	"strconv"
	"github.com/orbs-network/orbs-network-go/blockstorage"
	"github.com/orbs-network/orbs-network-go/instrumentation"
	"fmt"
	"context"
	"github.com/orbs-network/orbs-network-go/test/harness/gossip"
	"github.com/orbs-network/orbs-network-go/config"
)

type HttpServer interface {
	Stop()
}

type httpServer struct {
	httpServer *http.Server
}

func NewHttpServer(address string, nodeId string, isLeader bool, networkSize uint32) HttpServer {
	transport := gossip.NewPausableTransport()
	storage := blockstorage.NewInMemoryBlockPersistence(nodeId)
	logger := instrumentation.NewStdoutLog()
	lc := instrumentation.NewSimpleLoop(logger)
	nodeConfig := config.NewHardCodedConfig(networkSize)

	node := NewNode(transport, storage, logger, lc, nodeConfig, isLeader)

	server := &httpServer{
		httpServer: &http.Server {
			Addr:    address,
			Handler: createRouter(node.GetPublicApi()),
		},
	}

	go func() {
		server.httpServer.ListenAndServe() //TODO error on failed startup
	}()

	logger.Info(fmt.Sprintf("server started on address %s", address))

	return server

}

func createRouter(publicApi publicapi.PublicApi) http.Handler {
	sendTransactionHandler := func(w http.ResponseWriter, r *http.Request) {
		amountParam := r.URL.Query()["amount"][0]
		amount, _ := strconv.ParseInt(amountParam, 10, 32)
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

func (s *httpServer) Stop() {
	s.httpServer.Shutdown(context.TODO()) //TODO context
}
