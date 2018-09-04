package synchronization

import "time"

type TempUntilJonathanTrigger interface {
	Reset(duration time.Duration)
	Cancel()
}

type periodicalTrigger struct {
	timer *time.Timer
}

// empty implementation - WIP
func TempUntilJonathanTimer(interval time.Duration, trigger func()) TempUntilJonathanTrigger {
	t := &periodicalTrigger{
		timer: time.AfterFunc(interval, trigger),
	}
	return t
}

func (t *periodicalTrigger) Reset(duration time.Duration) {
	t.timer.Stop()
	t.timer.Reset(duration)
}

func (t *periodicalTrigger) Cancel() {
	t.timer.Stop()
}
