package wait

import (
	"github.com/pkg/errors"
	"time"
)

// useful for waiting on signal channels during tests
func AtMost(ch chan struct{}, timeout time.Duration) error {
	select {
	case <-ch:
		return nil
	case <-time.After(timeout):
		return errors.Errorf("no message in channel within %s", timeout)
	}
}

// nice wrapper for AtMost providing a default where we don't care about times
func ForSignal(ch chan struct{}) error {
	return AtMost(ch, 1*time.Second)
}
