package common

import (
	"time"

	"github.com/pkg/errors"
)

// ConditionFunc returns true if the condition is satisfied, or an error
// if the loop should be aborted.
type ConditionFunc func() (done bool, err error)

// SleepUntil waits for condition to succeeds.
func SleepUntil(backoff Backoff, condition ConditionFunc) error {
	var err error
	for backoff.MaxRetries > 0 {
		var ok bool
		if ok, err = condition(); ok {
			return err
		}
		if backoff.MaxRetries == 1 {
			break
		}
		backoff.MaxRetries--
		time.Sleep(backoff.Duration)

	}
	if err != nil {
		return errors.Wrap(err, "retrying timed out")
	}
	return errors.New("Timed out waiting for the condition")
}
