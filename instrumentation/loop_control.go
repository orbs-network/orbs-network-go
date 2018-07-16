package instrumentation

type LoopControl interface {
	NewLoop(name string, tickFunc func())
}

func NewSimpleLoop(events Reporting) LoopControl {
	return &simpleLoop{events: events}
}

type simpleLoop struct {
	events Reporting
}

func (l *simpleLoop) NewLoop(name string, tickFunc func()) {
	for {
		l.events.Info("enter_" + name)
		tickFunc()
		l.events.Info("exit_" + name)
	}
}
