package mocks

import (
	"time"
)

// Clock is a custom implementation of Time.
type Clock struct{}

// Now is a static never changing Time.
func (Clock) Now() time.Time {
	return time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
}

// After is what it was before.
func (Clock) After(d time.Duration) <-chan time.Time { return time.After(d) }
