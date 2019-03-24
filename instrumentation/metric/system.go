// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package metric

import (
	"context"
	"fmt"
	"github.com/c9s/goprocinfo/linux"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"github.com/orbs-network/orbs-network-go/synchronization"
	"os"
	"time"
)

type systemMetrics struct {
	rssBytes       *Gauge
	cpuUtilization *Gauge
}

type systemReporter struct {
	metrics systemMetrics
}

func NewSystemReporter(ctx context.Context, metricFactory Factory, logger log.BasicLogger) interface{} {
	r := &systemReporter{
		metrics: systemMetrics{
			rssBytes:       metricFactory.NewGauge("OS.Process.Memory.Bytes"),
			cpuUtilization: metricFactory.NewGauge("OS.Process.CPU.PerCent"),
		},
	}

	r.startReporting(ctx, logger)
	return r
}

func (r *systemReporter) startReporting(ctx context.Context, logger log.BasicLogger) {
	synchronization.NewPeriodicalTrigger(ctx, 3*time.Second, logger, func() {
		r.reportSystemMetrics(logger)
	}, nil)
}

const PAGESIZE = 4096

func (r *systemReporter) reportSystemMetrics(logger log.BasicLogger) {
	if _, err := os.Stat("/proc"); os.IsNotExist(err) {
		return
	}

	if rss, err := getRssMemory(); err != nil {
		logger.Error("failed to retrieve memory stats", log.Error(err))
	} else {
		r.metrics.rssBytes.Update(rss)
	}

	if cpu, err := getCPUUtilization(); err != nil {
		logger.Error("failed to retrieve cpu stats", log.Error(err))
	} else {
		r.metrics.cpuUtilization.Update(cpu)
	}

}

func getRssMemory() (int64, error) {
	statm, err := linux.ReadProcessStatm(fmt.Sprintf("/proc/%d/statm", os.Getpid()))
	if err != nil {
		return 0, err
	}

	return int64(statm.Resident * PAGESIZE), nil
}

func getCPUStats() (uint64, error) {
	cpu, err := linux.ReadStat("/proc/stat")
	if err != nil {
		return 0, err
	}
	e := cpu.CPUStatAll
	return e.User + e.Nice + e.System + e.Idle, nil
}

/**

https://github.com/Leo-G/DevopsWiki/wiki/How-Linux-CPU-Usage-Time-and-Percentage-is-calculated

Because the values reported by procfs are reported since startup, you have to sample twice and get the difference to gain meaningful insight.

For example, all processor cycles since the startup were reported as 1000 (cpu1), and after we check the second time 2000 (cpu2).

In the meantime, if we sample the process time as 100 (process1) and 200 (process2) respectively, we can calculate
the amount of cycles was spend on this particular process:

	percent = (process2 - process1) / (cpu2 - cpu1) * 100 = (200 - 100) / (2000 - 1000) * 100 = 100 / 1000 * 100 = 10

Here we take an easy way and sample cpu stats for all available processors and don't differentiate between cores,
and also don't take into account any cpu limits if they were applied and if it affected the procfs reporting.

*/

func getCPUUtilization() (int64, error) {
	pid := uint64(os.Getpid())

	firstSample, err := linux.ReadProcess(pid, "/proc")
	if err != nil {
		return 0, err
	}

	cpu1, err := getCPUStats()
	if err != nil {
		return 0, err
	}
	<-time.After(time.Second)
	secondSample, _ := linux.ReadProcess(pid, "/proc")

	user := (int64(secondSample.Stat.Utime) + secondSample.Stat.Cutime) - (int64(firstSample.Stat.Utime) + firstSample.Stat.Cutime)
	system := (int64(secondSample.Stat.Stime) + secondSample.Stat.Cstime) - (int64(firstSample.Stat.Stime) + firstSample.Stat.Cstime)
	cpu2, err := getCPUStats()
	if err != nil {
		return 0, err
	}

	percent := (float64(user+system) / float64(cpu2-cpu1)) * 100

	return int64(percent), nil
}
