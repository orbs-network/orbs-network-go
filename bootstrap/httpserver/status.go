package httpserver

import (
	"encoding/json"
	"github.com/orbs-network/scribe/log"
	"github.com/pkg/errors"
	"net/http"
	"time"
)

type StatusResponse struct {
	Timestamp time.Time
	Status    string
	Error     string `json:",omitempty"`
	Payload   interface{}
}

func (s *HttpServer) getStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	statusString := s.getStatusWarningMessage()
	var errorString string
	if statusString != "OK" {
		errorString = statusString
	}

	status := StatusResponse{
		Timestamp: time.Now(),
		Status:    statusString,
		Error:     errorString,
		Payload:   s.metricRegistry.ExportAllNested(),
	}

	data, _ := json.MarshalIndent(status, "", "\t")

	_, err := w.Write(data)
	if err != nil {
		s.logger.Info("error writing index.json response", log.Error(err))

	}

}

func (s *HttpServer) getStatusWarningMessage() string {
	maxTimeSinceLastBlockStorageUpdate := s.config.TransactionPoolTimeBetweenEmptyBlocks().Nanoseconds() * 10
	if maxTimeSinceLastBlockStorageUpdate < 600000000 { // ten minutes
		maxTimeSinceLastBlockStorageUpdate = 600000000
	}
	if lastBlockStorageUpdateTime, err := s.getGaugeValueFromMetrics("BlockStorage.FileSystemIndex.LastUpdateTime"); err == nil {
		if lastBlockStorageUpdateTime+maxTimeSinceLastBlockStorageUpdate < time.Now().UnixNano() {
			return "Last successful blockstorage update (including index update on boot) was too long ago"
		}
	}

	if len(s.config.ManagementFilePath()) != 0 && s.config.ManagementPollingInterval() > 0 {
		maxIntervalSinceLastSuccessfulManagementUpdate := int64(s.config.ManagementPollingInterval().Seconds()) * 20
		if managementLastSuccessfullUpdate, err := s.getGaugeValueFromMetrics("Management.Data.LastSuccessfulUpdateTime"); err == nil {
			if managementLastSuccessfullUpdate == 0 {
				return "Management Service has never successfully updated"
			}
			if managementLastSuccessfullUpdate+maxIntervalSinceLastSuccessfulManagementUpdate < time.Now().Unix() {
				return "Last successful Management Service update was too long ago"
			}
		}
	}

	return "OK"
}

func (s *HttpServer) getGaugeValueFromMetrics(name string) (int64, error) {
	metricObj := s.metricRegistry.Get(name)
	if metricObj == nil {
		return 0, errors.New("error retrieving metric value")
	}
	if value, ok := metricObj.Value().(int64); !ok {
		return 0, errors.New("error retrieving metric value")
	} else {
		return value, nil
	}
}
