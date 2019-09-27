package mocks

import (
	"time"
)

type Clock struct{}

func (Clock) Now() time.Time {
	return time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)

}

func (Clock) After(d time.Duration) <-chan time.Time { return time.After(d) }
