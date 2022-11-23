package common

import (
	"fmt"
	"time"
)

func ExampleSleepUntil() {
	// retry once in second, maximum retries 3 times
	backoff := Backoff{
		Duration:   1 * time.Second,
		MaxRetries: 3,
	}
	err := SleepUntil(backoff, func() (done bool, err error) {
		// will continue retrying
		return false, nil
		// return true, nil, exit immediately, should be used when ConditionFunc succeed
		// return false, err, exit immediately, should be used when ConditionFunc returns err that we should not retry anymore
	})
	fmt.Println(err.Error())
	// Output: Timed out waiting for the condition
}
