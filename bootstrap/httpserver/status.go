package httpserver

import (
	"encoding/json"
	"fmt"
	"github.com/orbs-network/scribe/log"
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
		Payload:   s.metricRegistry.ExportAllNested(s.logger),
	}

	data, _ := json.MarshalIndent(status, "", "\t")

	_, err := w.Write(data)
	if err != nil {
		s.logger.Info("error writing index.json response", log.Error(err))
	}
}

func (s *HttpServer) getStatusWarningMessage() string {
	var status string

	// if no genesis exists nothing to do
	if metricObj := s.metricRegistry.Get("Management.Data.GenesisRefTime"); metricObj != nil {
		genesisRefTime := s.getGaugeValueFromMetrics("Management.Data.GenesisRefTime")
		currentRefTime := s.getGaugeValueFromMetrics("Management.Data.CurrentRefTime")
		if (genesisRefTime > 0 && currentRefTime > genesisRefTime) || genesisRefTime == 0 { // blocks should be closing
			if metricObj := s.metricRegistry.Get("BlockStorage.FileSystemIndex.LastUpdateTime"); metricObj != nil {
				graceTimeSinceLastBlockStorageUpdate := s.config.TransactionPoolTimeBetweenEmptyBlocks() * 10
				if graceTimeSinceLastBlockStorageUpdate < time.Minute*10 {
					graceTimeSinceLastBlockStorageUpdate = time.Minute * 10
				}
				lastBlockStorageUpdateTime := s.getGaugeValueFromMetrics("BlockStorage.FileSystemIndex.LastUpdateTime")
				if lastBlockStorageUpdateTime+int64(graceTimeSinceLastBlockStorageUpdate.Seconds()) < time.Now().Unix() {
					status += fmt.Sprintf("Last successful blockstorage update:  %v,  was too long ago (last update includes indexing on boot) ;", lastBlockStorageUpdateTime)
				}
			}
		}

		if metricObj := s.metricRegistry.Get("Management.Data.LastSuccessfulUpdateTime"); metricObj != nil {
			if len(s.config.ManagementFilePath()) != 0 && s.config.ManagementPollingInterval() > 0 {
				graceIntervalSinceLastSuccessfulManagementUpdate := int64(s.config.ManagementPollingInterval().Seconds()) * 20
				managementLastSuccessfulUpdate := s.getGaugeValueFromMetrics("Management.Data.LastSuccessfulUpdateTime")
				if managementLastSuccessfulUpdate == 0 {
					status += "Management Service has never successfully updated ;"
				} else if managementLastSuccessfulUpdate+graceIntervalSinceLastSuccessfulManagementUpdate < time.Now().Unix() {
					status += fmt.Sprintf("Last successful Management Service update:  %v, was too long ago ;", managementLastSuccessfulUpdate)
				}
			}
		}
	}

	if status != "" {
		s.logger.Error("vc status issue ", log.String("status", status))
		return status
	}

	return "OK"
}

func (s *HttpServer) getGaugeValueFromMetrics(name string) int64 {
	metricObj := s.metricRegistry.Get(name)
	if metricObj == nil {
		return 0
	}
	if value, ok := metricObj.Value().(int64); !ok {
		return 0
	} else {
		return value
	}
}
