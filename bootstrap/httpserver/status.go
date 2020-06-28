package httpserver

import (
	"encoding/json"
	"github.com/orbs-network/orbs-network-go/config"
	"github.com/orbs-network/orbs-network-go/instrumentation/metric"
	"github.com/orbs-network/scribe/log"
	"net/http"
)

type StatusResponse struct {
	Uptime int64

	BlockHeight struct {
		BlockStorage int64
		StateStorage int64
		Timestamp    int64
	}

	Gossip struct {
		IncomingConnections int64
		OutgoingConnections int64
	}

	Management struct {
		LastUpdateTime int64
		Subscription   string
	}

	Version config.Version
}

func (s *HttpServer) getStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	metrics := s.metricRegistry

	data, _ := json.MarshalIndent(StatusResponse{
		Uptime: metricGetGaugeValue(s.logger, metrics, "Runtime.Uptime.Seconds"),

		BlockHeight: struct {
			BlockStorage int64
			StateStorage int64
			Timestamp    int64
		}{
			BlockStorage: metricGetGaugeValue(s.logger, metrics, "BlockStorage.BlockHeight"),
			StateStorage: metricGetGaugeValue(s.logger, metrics, "StateStorage.BlockHeight"),
			Timestamp:    metricGetGaugeValue(s.logger, metrics, "BlockStorage.LastCommitted.TimeNano"),
		},

		Gossip: struct {
			IncomingConnections int64
			OutgoingConnections int64
		}{
			IncomingConnections: metricGetGaugeValue(s.logger, metrics, "Gossip.IncomingConnection.Active.Count"),
			OutgoingConnections: metricGetGaugeValue(s.logger, metrics, "Gossip.OutgoingConnection.Active.Count"),
		},

		Management: struct {
			LastUpdateTime int64
			Subscription   string
		}{
			LastUpdateTime: metricGetGaugeValue(s.logger, metrics, "Management.LastUpdateTime"),
			Subscription:   metricGetString(s.logger, metrics, "Management.Subscription.Current"),
		},

		Version: config.GetVersion(),
	}, "", "  ")

	_, err := w.Write(data)
	if err != nil {
		s.logger.Info("error writing index.json response", log.Error(err))

	}

}

func metricGetGaugeValue(logger log.Logger, metrics metric.Registry, name string) (value int64) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("could not retrieve metric", log.String("metric", name))
		}
	}()

	rows := metrics.Get(name).Export().LogRow()
	value = rows[len(rows)-1].Int
	return value
}

func metricGetString(logger log.Logger, metrics metric.Registry, name string) (value string) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("could not retrieve metric", log.String("metric", name))
		}
	}()

	rows := metrics.Get(name).Export().LogRow()
	value = rows[len(rows)-1].StringVal
	return
}
