package utils

import (
	"time"
)

// Retriable is returned by RetriableFunction and tells whether to retry the function or not.
type Retriable bool

// RetriableFunction is function that can be retried
type RetriableFunction func() (Retriable, error)

// Retry retries retriableFunction for totalRetryCount times with a gap of retryPause.
// if retriableFunction returns boolean as false, then Retry will not retry and return error
// if retriableFunction returns boolean as true, then Retry will retry if fn returned an error
func Retry(totalRetryCount int, retryPause time.Duration, retriableFunction RetriableFunction) (err error) {
	retryCounter := 0
	retry := Retriable(true)
	for {
		retry, err = retriableFunction()
		if err == nil || !retry {
			break
		}

		retryCounter++
		if retryCounter >= totalRetryCount {
			break
		}
		time.Sleep(retryPause)
	}
	return
}
