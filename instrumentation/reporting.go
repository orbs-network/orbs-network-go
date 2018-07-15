package instrumentation

import (
	"log"
)

type Reporting interface {
	Info(message string)
	Infof(message string, parmams ...interface{})
	Error(err error)
}

const FinishedConsensusRound = "finished_consensus_round"
const ConsensusError = "consensus_error"

type StdoutLog interface {
	Reporting
}

type stdoutLog struct {
}

func NewStdoutLog() Reporting {
	return &stdoutLog{}
}

func (e *stdoutLog) Info(message string) {
	log.Print(message)
}

func (e *stdoutLog) Infof(message string, params ...interface{}) {
	log.Printf(message, params...)
}

func (e *stdoutLog) Error(err error) {
	log.Fatal(err)
}

type compositeReporting struct {
	children []Reporting
}

func NewCompositeReporting(children []Reporting) Reporting {
	return &compositeReporting{children: children}
}

func (e *compositeReporting) Info(message string) {
	for _, child := range e.children {
		child.Info(message)
	}
}

func (e *compositeReporting) Infof(message string, params ...interface{}) {
	for _, child := range e.children {
		child.Infof(message, params...)
	}
}

func (e *compositeReporting) Error(err error) {
	for _, child := range e.children {
		child.Error(err)
	}
}
