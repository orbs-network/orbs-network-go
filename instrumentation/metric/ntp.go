package metric

import (
	"context"
	"github.com/beevik/ntp"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"github.com/orbs-network/scribe/log"
	"time"
)

type ntpMetrics struct {
	drift *Gauge
}

type ntpReporter struct {
	metrics ntpMetrics
	address string
}

const NTP_QUERY_INTERVAL = 30 * time.Second

func NewNtpReporter(ctx context.Context, metricFactory Factory, logger log.Logger, ntpServerAddress string) interface{} {
	r := &ntpReporter{
		metrics: ntpMetrics{
			drift: metricFactory.NewGauge("OS.Time.Drift.Millis"),
		},
		address: ntpServerAddress,
	}

	if ntpServerAddress != "" {
		r.startReporting(ctx, logger)
	}

	return r
}

func (r *ntpReporter) startReporting(ctx context.Context, logger log.Logger) {
	synchronization.NewPeriodicalTrigger(ctx, NTP_QUERY_INTERVAL, logger, func() {
		response, err := ntp.Query(r.address)

		if err != nil {
			logger.Info("could not query ntp server", log.String("ntp-server", r.address))
		} else {
			driftInMillis := response.ClockOffset.Nanoseconds() / 1000000
			r.metrics.drift.Update(driftInMillis)
		}
	}, nil)
}
