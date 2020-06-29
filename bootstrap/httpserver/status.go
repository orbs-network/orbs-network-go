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

	BlockStorage_BlockHeight   int64
	StateStorage_BlockHeight   int64
	BlockStorage_LastCommitted int64

	Gossip_IncomingConnections int64
	Gossip_OutgoingConnections int64

	Management_LastUpdated  int64
	Management_Subscription string

	Version config.Version
}

func (s *HttpServer) getStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	metrics := s.metricRegistry

	data, _ := json.MarshalIndent(StatusResponse{
		Uptime: metricGetGaugeValue(s.logger, metrics, "Runtime.Uptime.Seconds"),

		BlockStorage_BlockHeight:   metricGetGaugeValue(s.logger, metrics, "BlockStorage.BlockHeight"),
		StateStorage_BlockHeight:   metricGetGaugeValue(s.logger, metrics, "StateStorage.BlockHeight"),
		BlockStorage_LastCommitted: metricGetGaugeValue(s.logger, metrics, "BlockStorage.LastCommitted.TimeNano"),

		Gossip_IncomingConnections: metricGetGaugeValue(s.logger, metrics, "Gossip.IncomingConnection.Active.Count"),
		Gossip_OutgoingConnections: metricGetGaugeValue(s.logger, metrics, "Gossip.OutgoingConnection.Active.Count"),

		Management_LastUpdated:  metricGetGaugeValue(s.logger, metrics, "Management.LastUpdateTime"),
		Management_Subscription: metricGetString(s.logger, metrics, "Management.Subscription.Current"),

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
