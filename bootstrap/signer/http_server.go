package signer

import (
	"context"
	"github.com/orbs-network/orbs-network-go/bootstrap/httpserver"
	"github.com/orbs-network/scribe/log"
	"net"
	"net/http"
	"time"
)

type httpServer struct {
	server *http.Server
	port   int
	logger log.Logger
}

// TODO: unify with httpserver.HttpServer
func NewHttpServer(address string, logger log.Logger, setup func(router *http.ServeMux)) (httpserver.HttpServer, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	logger.Info("started http server", log.String("address", address))

	router := http.NewServeMux()
	setup(router)

	s := &httpServer{
		server: &http.Server{
			Handler: router,
		},
		port:   listener.Addr().(*net.TCPAddr).Port,
		logger: logger,
	}

	// We prefer not to use `HttpServer.ListenAndServe` because we want to block until the socket is listening or exit immediately
	go s.server.Serve(httpserver.TcpKeepAliveListener{listener.(*net.TCPListener)})

	return s, nil
}

func (s *httpServer) GracefulShutdown(timeout time.Duration) {
	ctx := context.Background()
	if timeout > 0 {
		var cancel func()
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	if err := s.server.Shutdown(ctx); err != nil {
		s.logger.Error("failed to stop http server gracefully", log.Error(err))
	}
}

func (s *httpServer) Port() int {
	return s.port
}
