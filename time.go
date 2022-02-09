package common

import (
	"time"

	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/util/wait"
)

// SleepUntil waits for condition to succees
func SleepUntil(backoff wait.Backoff, condition wait.ConditionFunc) error {
	var err error
	for backoff.Steps > 0 {
		var ok bool
		if ok, err = condition(); ok {
			return err
		}
		if backoff.Steps == 1 {
			break
		}
		backoff.Steps--
		time.Sleep(backoff.Duration)

	}
	if err != nil {
		return errors.Wrap(err, "retrying timed out")
	}
	return errors.New("Timed out waiting for the condition")
}
