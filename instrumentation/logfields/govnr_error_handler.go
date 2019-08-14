package logfields

import (
	"github.com/orbs-network/govnr"
	"github.com/orbs-network/scribe/log"
	"runtime/debug"
)

type Errorer interface {
	Error(string, ...*log.Field)
}

type govnrErrorer struct {
	logger Errorer
}

func (h *govnrErrorer) Error(err error) {
	h.logger.Error("recovered panic", log.Error(err), log.String("panic", "true"), log.String("stack-trace", string(debug.Stack())))
}

func GovnrErrorer(logger Errorer) govnr.Errorer {
	return &govnrErrorer{logger}
}
