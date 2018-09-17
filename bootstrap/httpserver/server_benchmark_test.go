package httpserver

import (
	"bytes"
	"github.com/orbs-network/go-mock"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-spec/types/go/primitives"
	"github.com/orbs-network/orbs-spec/types/go/protocol"
	"github.com/orbs-network/orbs-spec/types/go/protocol/client"
	"github.com/orbs-network/orbs-spec/types/go/services"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
	"time"
)

func BenchmarkServerCallMethod(b *testing.B) {
	s := startServer()
	s.GracefulShutdown(time.Second)

	webClient := &http.Client{}

	request := (&client.CallMethodRequestBuilder{
		Transaction: &protocol.TransactionBuilder{},
	}).Build()

	req, _ := http.NewRequest("POST", "http://127.0.0.1:8080/api/v1/call-method", bytes.NewReader(request.Raw()))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sendRequest(webClient, req)
	}
}

func BenchmarkFastServerCallMethod(b *testing.B) {
	s := startFastServer()
	s.GracefulShutdown(time.Second)

	webClient := &http.Client{}

	request := (&client.CallMethodRequestBuilder{
		Transaction: &protocol.TransactionBuilder{},
	}).Build()

	req, _ := http.NewRequest("POST", "http://127.0.0.1:8080/api/v1/call-method", bytes.NewReader(request.Raw()))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sendRequest(webClient, req)
	}
}

func startServer() HttpServer {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	papiMock := &services.MockPublicApi{}
	response := &client.CallMethodResponseBuilder{
		RequestStatus:       protocol.REQUEST_STATUS_COMPLETED,
		OutputArgumentArray: nil,
		CallMethodResult:    protocol.EXECUTION_RESULT_SUCCESS,
		BlockHeight:         1,
		BlockTimestamp:      primitives.TimestampNano(time.Now().Nanosecond()),
	}

	papiMock.When("CallMethod", mock.Any).Times(1).Return(&services.CallMethodOutput{ClientResponse: response.Build()})

	return NewHttpServer("127.0.0.1:8080", logger, papiMock)
}

func startFastServer() HttpServer {
	logger := log.GetLogger().WithOutput(log.NewOutput(os.Stdout).WithFormatter(log.NewHumanReadableFormatter()))
	papiMock := &services.MockPublicApi{}
	response := &client.CallMethodResponseBuilder{
		RequestStatus:       protocol.REQUEST_STATUS_COMPLETED,
		OutputArgumentArray: nil,
		CallMethodResult:    protocol.EXECUTION_RESULT_SUCCESS,
		BlockHeight:         1,
		BlockTimestamp:      primitives.TimestampNano(time.Now().Nanosecond()),
	}

	papiMock.When("CallMethod", mock.Any).Times(1).Return(&services.CallMethodOutput{ClientResponse: response.Build()})

	return NewFastHttpServer("127.0.0.1:8080", logger, papiMock)
}

func sendRequest(client *http.Client, request *http.Request) {
	res, err := client.Do(request)
	if err != nil {
		panic(err)
	}

	if res.StatusCode != 200 {
		panic("request failed")
	}

	_, err = ioutil.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	err = res.Body.Close()
	if err != nil {
		panic(err)
	}
}
