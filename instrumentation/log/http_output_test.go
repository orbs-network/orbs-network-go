package log

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"io/ioutil"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

type httpOutputHarness struct {
	port     uint16
	listener net.Listener
	router   *http.ServeMux
}

func newHttpHarness(handler http.Handler) *httpOutputHarness {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	port := uint16(20000 + r.Intn(40000))

	router := http.NewServeMux()
	router.Handle("/submit-logs", handler)

	return &httpOutputHarness{
		port:   port,
		router: router,
	}
}

func (h *httpOutputHarness) start(t *testing.T) {
	go func() {
		address := fmt.Sprintf("0.0.0.0:%d", h.port)
		t.Log("Serving http requests on", address)

		listener, err := net.Listen("tcp", address)
		h.listener = listener

		require.NoError(t, err, "failed to use http port")

		err = http.Serve(listener, h.router)
		require.NoError(t, err, "failed to serve http requests")
	}()
}

func (h *httpOutputHarness) stop() {
	if h.listener != nil {
		h.listener.Close()
	}
}

func TestHttpWriter_Write(t *testing.T) {
	h := newHttpHarness(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()

		require.EqualValues(t, []byte("hello"), body)

		w.WriteHeader(200)
	}))
	h.start(t)
	defer h.stop()

	w := NewHttpWriter(fmt.Sprintf("http://localhost:%d/submit-logs", h.port))
	size, err := w.Write([]byte("hello"))
	require.NoError(t, err)
	require.EqualValues(t, 5, size)
}

func TestHttpOutput_Append(t *testing.T) {
	h := newHttpHarness(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := ioutil.ReadAll(r.Body)
		defer r.Body.Close()

		lines := strings.Split(string(body), "\n")
		require.EqualValues(t, 4, len(lines))

		w.WriteHeader(200)
	}))
	h.start(t)
	defer h.stop()

	logger := GetLogger().WithOutput(
		NewHttpOutput(
			NewHttpWriter(fmt.Sprintf("http://localhost:%d/submit-logs", h.port)),
			NewJsonFormatter(),
			10000,
			time.Microsecond))

	logger.Info("Ground control to Major Tom")
	logger.Info("Commencing countdown")
	logger.Info("Engines on")

	time.Sleep(2 * time.Millisecond)
}
