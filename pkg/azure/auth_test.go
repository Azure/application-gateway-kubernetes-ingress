// +build unittest

package azure

import (
	"os"
	"time"

	"github.com/Azure/go-autorest/autorest/azure/auth"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/klog/v2"
)

var _ = Describe("Auth Tests", func() {
	When("test getAuthorizer fails", func() {
		BeforeEach(func() {
			os.Setenv(auth.ClientID, "invalid-client-id")
			os.Unsetenv(auth.TenantID)
			os.Unsetenv(auth.ClientSecret)
		})
		It("getAuthorizer should try and get some authorizer but fail", func() {
			authorizer, err := getAuthorizer("", false, nil)
			klog.Info(authorizer)
			Ω(err).To(HaveOccurred())
			Ω(authorizer).To(BeNil())
		})
		It("getAuthorizerWithRetry should try and get some authorizer but fail", func() {
			authorizer, err := GetAuthorizerWithRetry("", false, nil, 0, time.Duration(10))
			klog.Info(authorizer)
			Ω(err).To(HaveOccurred())
			Ω(authorizer).To(BeNil())
		})
	})
	When("authorizer succeeds", func() {
		BeforeEach(func() {
			os.Setenv(auth.ClientID, "guid1")
			os.Setenv(auth.TenantID, "guid2")
			os.Setenv(auth.ClientSecret, "fake-secret")
		})
		It("getAuthorizer should try and get some authorizer", func() {
			authorizer, err := getAuthorizer("", false, nil)
			Ω(err).ToNot(HaveOccurred())
			Ω(authorizer).ToNot(BeNil())
		})

		It("getAuthorizerWithRetry should try and get some authorizer", func() {
			authorizer, err := GetAuthorizerWithRetry("", false, nil, 0, time.Duration(10))
			Ω(err).ToNot(HaveOccurred())
			Ω(authorizer).ToNot(BeNil())
		})

		AfterEach(func() {
			os.Unsetenv(auth.ClientID)
			os.Unsetenv(auth.TenantID)
			os.Unsetenv(auth.ClientSecret)
		})
	})

})
