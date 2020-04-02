// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package environment

import (
	"os"
	"regexp"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestEnvironment(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Environment Suite")
}

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
				Expect(GetEnvironmentVariable("-non-existent-key-we-hope", "expected-value", nil)).To(Equal("expected-value"))
			})
			It("returns expected value", func() {
				defaultValue := "--default--value--"
				passingValidator := regexp.MustCompile(`^[a-zA-Z\-]+$`)
				Expect(GetEnvironmentVariable(envVar, defaultValue, passingValidator)).To(Equal("expected-value"))
			})
			It("returns default value in absence of an env var", func() {
				defaultValue := "--default--value--"
				// without a validator we get the environment variable's value
				Expect(GetEnvironmentVariable(envVar, defaultValue, nil)).To(Equal(expectedEnvVarValue))

				// with a non-passing validator we get the default value
				failingValidator := regexp.MustCompile(`^[0-9]+$`)
				Expect(GetEnvironmentVariable(envVar, defaultValue, failingValidator)).To(Equal(defaultValue))
			})

			It("GetEnv returns struct", func() {
				_ = os.Setenv(SubscriptionIDVarName, "SubscriptionIDVarName")
				_ = os.Setenv(ResourceGroupNameVarName, "ResourceGroupNameVarName")
				_ = os.Setenv(AppGwNameVarName, "AppGwNameVarName")
				_ = os.Setenv(AuthLocationVarName, "AuthLocationVarName")
				_ = os.Setenv(WatchNamespaceVarName, "WatchNamespaceVarName")
				_ = os.Setenv(UsePrivateIPVarName, "UsePrivateIPVarName")
				_ = os.Setenv(VerbosityLevelVarName, "VerbosityLevelVarName")
				_ = os.Setenv(EnableBrownfieldDeploymentVarName, "SomethingIrrelevant1234")
				_ = os.Setenv(EnableIstioIntegrationVarName, "true")
				_ = os.Setenv(EnableSaveConfigToFileVarName, "false")
				_ = os.Setenv(EnablePanicOnPutErrorVarName, "true")
				_ = os.Setenv(ReconcilePeriodSecondsVarName, "30")

				expected := EnvVariables{
					SubscriptionID:             "SubscriptionIDVarName",
					ResourceGroupName:          "ResourceGroupNameVarName",
					AppGwName:                  "AppGwNameVarName",
					AuthLocation:               "AuthLocationVarName",
					WatchNamespace:             "WatchNamespaceVarName",
					UsePrivateIP:               "UsePrivateIPVarName",
					VerbosityLevel:             "VerbosityLevelVarName",
					EnableBrownfieldDeployment: false,
					EnableIstioIntegration:     true,
					EnableSaveConfigToFile:     false,
					EnablePanicOnPutError:      true,
					HTTPServicePort:            "8123",
					ReconcilePeriodSeconds:     "30",
				}

				Expect(GetEnv()).To(Equal(expected))
			})
		})

		Context("Test ValidateEnv when APPGW_ENABLE_DEPLOY is FALSE", func() {
			It("should throw error when neither applicationGatewayName or applicationGatewayID is passed when APPGW_ENABLE_DEPLOY is FALSE", func() {
				env := EnvVariables{
					EnableDeployAppGateway: false,
				}
				Expect(ValidateEnv(env)).To(Equal(ErrorMissingApplicationGatewayNameOrApplicationGatewayID))
			})

			It("should allow passing applicationGatewayID when APPGW_ENABLE_DEPLOY is FALSE", func() {
				env := EnvVariables{
					AppGwResourceID:        "id",
					EnableDeployAppGateway: false,
				}
				Expect(ValidateEnv(env)).To(BeNil())
			})

			It("should allow passing applicationGatewayName when APPGW_ENABLE_DEPLOY is FALSE", func() {
				env := EnvVariables{
					AppGwName:              "name",
					EnableDeployAppGateway: false,
				}
				Expect(ValidateEnv(env)).To(BeNil())
			})
		})

		Context("Test ValidateEnv when APPGW_ENABLE_DEPLOY is TRUE", func() {
			It("should throw error when applicationGatewayName is missing when APPGW_ENABLE_DEPLOY is TRUE", func() {
				env := EnvVariables{
					EnableDeployAppGateway: true,
				}
				Expect(ValidateEnv(env)).To(Equal(ErrorMissingApplicationgatewayName))
			})

			It("should throw error when applicationGatewayID is provided when APPGW_ENABLE_DEPLOY is TRUE", func() {
				env := EnvVariables{
					AppGwResourceID:        "id",
					EnableDeployAppGateway: true,
				}
				Expect(ValidateEnv(env)).To(Equal(ErrorNotAllowedApplicationgatewayID))
			})

			It("should throw error when subnet info is missing when APPGW_ENABLE_DEPLOY is TRUE", func() {
				env := EnvVariables{
					AppGwName:              "name",
					EnableDeployAppGateway: true,
				}
				Expect(ValidateEnv(env)).To(Equal(ErrorMissingSubnetInfo))
			})

			It("should allow passing applicationGatewayName when APPGW_ENABLE_DEPLOY is TRUE", func() {
				env := EnvVariables{
					AppGwName:              "name",
					AppGwSubnetPrefix:      "prefix",
					EnableDeployAppGateway: true,
				}
				Expect(ValidateEnv(env)).To(BeNil())
			})

			It("should allow passing applicationGatewayName when APPGW_ENABLE_DEPLOY is TRUE", func() {
				env := EnvVariables{
					AppGwName:              "name",
					AppGwSubnetID:          "id",
					EnableDeployAppGateway: true,
				}
				Expect(ValidateEnv(env)).To(BeNil())
			})
		})

		Context("Test ValidateEnv for RECONCILE_PERIOD_SECONDS", func() {
			It("should error when invalid input in RECONCILE_PERIOD_SECONDS", func() {
				env := EnvVariables{
					AppGwResourceID:        "id",
					EnableDeployAppGateway: false,

					ReconcilePeriodSeconds: "string",
				}
				Expect(ValidateEnv(env)).To(Equal(ErrorInvalidReconcilePeriod))
			})

			It("should not error when input is in range in RECONCILE_PERIOD_SECONDS", func() {
				env := EnvVariables{
					AppGwResourceID:        "id",
					EnableDeployAppGateway: false,

					ReconcilePeriodSeconds: "30",
				}
				Expect(ValidateEnv(env)).To(BeNil())

				env.ReconcilePeriodSeconds = "300"
				Expect(ValidateEnv(env)).To(BeNil())
			})

			It("should error when input is out of range in RECONCILE_PERIOD_SECONDS", func() {
				env := EnvVariables{
					AppGwResourceID:        "id",
					EnableDeployAppGateway: false,

					ReconcilePeriodSeconds: "29",
				}
				Expect(ValidateEnv(env)).To(Equal(ErrorInvalidReconcilePeriod))

				env.ReconcilePeriodSeconds = "301"
				Expect(ValidateEnv(env)).To(Equal(ErrorInvalidReconcilePeriod))
			})
		})

	})
})
