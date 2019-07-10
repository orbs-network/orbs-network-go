package gamma

import (
	"net/http"
	"strconv"
	"time"
)

func (s *GammaServer) addGammaHandlers(router *http.ServeMux) {
	router.HandleFunc("/debug/gamma/shutdown", s.shutdownHandler)
	router.HandleFunc("/debug/gamma/inc-time", s.incrementTime)
}

func (s *GammaServer) shutdownHandler(writer http.ResponseWriter, request *http.Request) {
	if request.Method != http.MethodPost {
		writer.WriteHeader(http.StatusBadRequest)
		return
	}

	writer.Write([]byte(`{"status":"shutting down"}`))

	s.GracefulShutdown(1 * time.Second)
}

func (s *GammaServer) incrementTime(writer http.ResponseWriter, request *http.Request) {
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
