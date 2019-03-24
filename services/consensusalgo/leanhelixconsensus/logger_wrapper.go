// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package leanhelixconsensus

import (
	"fmt"
	"github.com/orbs-network/orbs-network-go/instrumentation/log"
	"strings"
)

type loggerWrapper struct {
	log       log.BasicLogger
	showDebug bool
}

const LH_PREFIX = "=LH= "

func NewLoggerWrapper(log log.BasicLogger, showDebug bool) *loggerWrapper {
	return &loggerWrapper{
		log:       log,
		showDebug: showDebug,
	}
}

func (l *loggerWrapper) ShowDebug(showDebug bool) {
	l.showDebug = showDebug
}

func (l *loggerWrapper) Debug(format string, args ...interface{}) {
	if !l.showDebug {
		return
	}
	str := strings.Join([]string{LH_PREFIX, format}, "")
	finalStr := fmt.Sprintf(str, args...)
	l.log.Info(finalStr, log.String("lhlib", "true"))
}

func (l *loggerWrapper) Info(format string, args ...interface{}) {
	l.log.Info(fmt.Sprintf(strings.Join([]string{LH_PREFIX, format}, ""), args...), log.String("lhlib", "true"))
}

func (l *loggerWrapper) Error(format string, args ...interface{}) {
	l.log.Error(fmt.Sprintf(strings.Join([]string{LH_PREFIX, format}, ""), args...), log.String("lhlib", "true"))
}
