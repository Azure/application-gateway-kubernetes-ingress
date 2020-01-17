package utils

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Retry", func() {
	Describe("Testing `retry` function", func() {
		Context("retry stop trying", func() {
			It("should not retry if no error.", func() {
				retry := 10
				counter := 0
				Retry(retry, time.Duration(0),
					func() (Retriable, error) {
						counter++
						return Retriable(true), nil
					})
				Expect(counter).To(Equal(1))
			})

			It("should not retry if function asks to not retry.", func() {
				retry := 10
				counter := 0
				err := errors.New("fake")
				retryError := Retry(retry, time.Duration(0),
					func() (Retriable, error) {
						counter++
						return Retriable(false), err
					})
				Expect(counter).To(Equal(1))
				Expect(err).To(Equal(retryError))
			})
		})

		Context("Test retry count", func() {
			It("should retry equal to retry count.", func() {
				retry := 10
				counter := 0
				err := errors.New("fake")
				retryError := Retry(retry, time.Duration(0),
					func() (Retriable, error) {
						counter++
						return Retriable(true), err
					})
				Expect(counter).To(Equal(retry))
				Expect(err).To(Equal(retryError))
			})
		})

		Context("Test retry pause", func() {
			It("should pause between every retry.", func() {
				retry := 2
				counter := 0
				err := errors.New("fake")
				pause := time.Second
				execTimeList := make([]time.Time, 0)
				retryError := Retry(retry, pause,
					func() (Retriable, error) {
						counter++
						execTimeList = append(execTimeList, time.Now())
						return Retriable(true), err
					})
				Expect(counter).To(Equal(2))
				Expect(err).To(Equal(retryError))
				timeGap := execTimeList[1].Sub(execTimeList[0])
				Expect(timeGap).To(BeNumerically(">", pause))
			})
		})
	})
})
