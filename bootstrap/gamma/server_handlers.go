package gamma

import (
	"github.com/orbs-network/orbs-network-go/synchronization/supervised"
	"net/http"
	"strconv"
	"time"
)

func (s *Server) addGammaHandlers(router *http.ServeMux) {
	router.HandleFunc("/debug/gamma/shutdown", s.shutdownHandler)
	router.HandleFunc("/debug/gamma/inc-time", s.incrementTime)
}

func (s *Server) shutdownHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	writer.Write([]byte(`{"status":"shutting down"}`))

	supervised.ShutdownGracefully(s, 1*time.Second)
}

func (s *Server) incrementTime(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	if err := request.ParseForm(); err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	secondsParam := request.Form.Get("seconds-to-add")
	if secondsParam == "" {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	seconds, err := strconv.Atoi(secondsParam)
	if err != nil {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	s.clock.AddSeconds(seconds)

}
