package environment_test

import (
	"regexp"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"os"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Environment", func() {
	Describe("Testing `Environment` helpers", func() {
		Context("Testing the GetEnv helper", func() {
			const (
				expectedEnvVarValue = "expected-value"
				envVar              = "---some--environment--variable--with-low-likelihood-that-will-collide---"
			)
			BeforeEach(func() {
				// Make sure the environment variable we are using for this test does not already exist in the OS.
				_, exists := os.LookupEnv(envVar)
				Expect(exists).To(BeFalse())
				// Set it
				_ = os.Setenv(envVar, expectedEnvVarValue)
				_, exists = os.LookupEnv(envVar)
				Expect(exists).To(BeTrue())
			})
			AfterEach(func() {
				// Clean up the env var after the tests are done
				_ = os.Unsetenv(envVar)
			})
			It("returns default value in absence of an env var", func() {
				Expect(environment.GetEnvironmentVariable("-non-existent-key-we-hope", "expected-value", nil)).To(Equal("expected-value"))
			})
			It("returns expected value", func() {
				defaultValue := "--default--value--"
				passingValidator := regexp.MustCompile(`^[a-zA-Z\-]+$`)
				Expect(environment.GetEnvironmentVariable(envVar, defaultValue, passingValidator)).To(Equal("expected-value"))
			})
			It("returns default value in absence of an env var", func() {
				defaultValue := "--default--value--"
				// without a validator we get the environment variable's value
				Expect(environment.GetEnvironmentVariable(envVar, defaultValue, nil)).To(Equal(expectedEnvVarValue))

				// with a non-passing validator we get the default value
				failingValidator := regexp.MustCompile(`^[0-9]+$`)
				Expect(environment.GetEnvironmentVariable(envVar, defaultValue, failingValidator)).To(Equal(defaultValue))
			})
		})
	})
})