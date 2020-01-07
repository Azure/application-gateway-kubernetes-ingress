package utils

import (
	"time"
)

// RetriableFunction is function that can be retried
type RetriableFunction func() (bool, error)

// Retry retries retriableFunction for totalRetryCount times with a gap of retryPause.
// if retriableFunction returns boolean as false, then Retry will not retry and return error
// if retriableFunction returns boolean as true, then Retry will retry if fn returned an error
func Retry(totalRetryCount int, retryPause time.Duration, retriableFunction RetriableFunction) (err error) {
	retryCounter := 0
	retry := true
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
