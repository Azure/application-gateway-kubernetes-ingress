package utils

import (
	"errors"
	"time"

	. "github.com/onsi/ginkgo/v2"
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
				Expect(retryError).To(Equal(err))
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
				Expect(retryError).To(Equal(err))
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
				Expect(retryError).To(Equal(err))
				timeGap := execTimeList[1].Sub(execTimeList[0])
				Expect(timeGap).To(BeNumerically(">", pause))
			})
		})

		Context("Test retry forever", func() {
			It("when retry count is -1 but should stop when retry false", func() {
				retry := -1
				counter := 0
				err := errors.New("fake")
				retryError := Retry(retry, time.Duration(0),
					func() (Retriable, error) {
						counter++
						if counter == 11 {
							return Retriable(false), err
						}
						return Retriable(true), err
					})
				Expect(counter).To(Equal(11))
				Expect(retryError).To(Equal(err))
			})

			It("when retry count is -1 but should stop when err is nil", func() {
				retry := -1
				counter := 0
				err := errors.New("fake")
				retryError := Retry(retry, time.Duration(0),
					func() (Retriable, error) {
						counter++
						if counter == 11 {
							return Retriable(true), nil
						}
						return Retriable(true), err
					})
				Expect(counter).To(Equal(11))
				Expect(retryError).To(BeNil())
			})
		})
	})
})
