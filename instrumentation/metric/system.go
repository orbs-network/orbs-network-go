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
			rssBytes:       metricFactory.NewGauge("System.Memory.Rss.Bytes"),
			cpuUtilization: metricFactory.NewGauge("System.CPU.PerCent"),
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
