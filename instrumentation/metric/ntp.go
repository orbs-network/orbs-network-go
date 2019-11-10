// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"context"
	"github.com/beevik/ntp"
	"github.com/orbs-network/govnr"
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

func NewNtpReporter(ctx context.Context, metricFactory Factory, logger log.Logger, ntpServerAddress string) govnr.ShutdownWaiter {
	r := &ntpReporter{
		metrics: ntpMetrics{
			drift: metricFactory.NewGauge("OS.Time.Drift.Millis"),
		},
		address: ntpServerAddress,
	}

	return r.startReporting(ctx, logger)
}

func (r *ntpReporter) startReporting(ctx context.Context, logger log.Logger) govnr.ShutdownWaiter {
	return synchronization.NewPeriodicalTrigger(ctx, "NTP metric reporter", synchronization.NewTimeTicker(NTP_QUERY_INTERVAL), logger, func() {
		response, err := ntp.Query(r.address)

		if err != nil {
			logger.Info("could not query ntp server", log.String("ntp-server", r.address))
		} else {
			driftInMillis := response.ClockOffset.Nanoseconds() / 1000000
			r.metrics.drift.Update(driftInMillis)
		}
	}, nil)
}
