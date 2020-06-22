package utils

import (
	"time"

	"github.com/golang/glog"
)

const (
	// RetryPause to pause 10 sec before next retry
	RetryPause = 10 * time.Second
	// RetryCount indicates the retry times
	RetryCount = 3
)

// Retriable is returned by RetriableFunction and tells whether to retry the function or not.
type Retriable bool

// RetriableFunction is function that can be retried
type RetriableFunction func() (Retriable, error)

// Retry retries retriableFunction for totalRetryCount times with a gap of retryPause.
// if retriableFunction returns boolean as false, then Retry will not retry and return error
// if retriableFunction returns boolean as true, then Retry will retry if fn returned an error
// if totalRetryCount is -1, then retry happen forever until one of the two above conditions are satisfied.
func Retry(totalRetryCount int, retryPause time.Duration, retriableFunction RetriableFunction) (err error) {
	retryCounter := 0
	retry := Retriable(true)
	for {
		retry, err = retriableFunction()
		if err == nil || !retry {
			break
		}

		retryCounter++
		if totalRetryCount != -1 && retryCounter >= totalRetryCount {
			break
		}

		glog.Infof("Retrying in %s", retryPause)
		time.Sleep(retryPause)
	}
	return
}
