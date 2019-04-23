// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

//+build manual linux

package metric

import (
	"context"
	"github.com/orbs-network/scribe/log"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestSystemMetrics(t *testing.T) {
	//stat, _ := linux.ReadProcessStat("../../vendor/github.com/c9s/goprocinfo/linux/proc/3323/stat")
	//fmt.Println(stat.Rss)
	//
	//

	go func() {
		ioutil.ReadFile("/dev/random")
	}()

	m := NewRegistry()
	l := log.GetLogger().WithOutput(log.NewFormattingOutput(os.Stderr, log.NewHumanReadableFormatter()))
	NewSystemReporter(context.Background(), m, l)
	m.PeriodicallyRotate(context.Background(), l)

	<-time.After(1 * time.Minute)
}
