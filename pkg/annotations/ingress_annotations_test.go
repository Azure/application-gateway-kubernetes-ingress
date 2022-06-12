// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

//go:build unittest
// +build unittest

package annotations

import (
	"fmt"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	networking "k8s.io/api/networking/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
)

var ingress = networking.Ingress{
	ObjectMeta: v1.ObjectMeta{
		Annotations: map[string]string{},
	},
}

const (
	NoError = "Expected to return %s and no error. Returned %v and %v."
	Error   = "Expected to return error %s. Returned %v and %v."
)

func TestIt(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Run All main.go Tests")
}

var _ = Describe("Test ingress annotation functions", func() {
	annotations := map[string]string{
		"appgw.ingress.kubernetes.io/use-private-ip":                      "true",
		"appgw.ingress.kubernetes.io/override-frontend-port":              "444",
		"appgw.ingress.kubernetes.io/connection-draining":                 "true",
		"appgw.ingress.kubernetes.io/cookie-based-affinity":               "true",
		"appgw.ingress.kubernetes.io/cookie-based-affinity-distinct-name": "true",
		"appgw.ingress.kubernetes.io/ssl-redirect":                        "true",
		"appgw.ingress.kubernetes.io/request-timeout":                     "123456",
		"appgw.ingress.kubernetes.io/connection-draining-timeout":         "3456",
		"appgw.ingress.kubernetes.io/backend-path-prefix":                 "prefix-here",
		"appgw.ingress.kubernetes.io/backend-hostname":                    "www.backend.com",
		"appgw.ingress.kubernetes.io/hostname-extension":                  "www.bye.com, www.b*.com",
		"appgw.ingress.kubernetes.io/appgw-ssl-certificate":               "appgw-cert",
		"appgw.ingress.kubernetes.io/appgw-ssl-profile":                   "legacy-tls",
		"appgw.ingress.kubernetes.io/appgw-trusted-root-certificate":      "appgw-root-cert1,appgw-root-cert2",
		"appgw.ingress.kubernetes.io/health-probe-hostname":               "myhost.mydomain.com",
		"appgw.ingress.kubernetes.io/health-probe-port":                   "8080",
		"appgw.ingress.kubernetes.io/health-probe-path":                   "/healthz",
		"appgw.ingress.kubernetes.io/health-probe-status-codes":           "200-399, 401",
		"appgw.ingress.kubernetes.io/health-probe-interval":               "15",
		"appgw.ingress.kubernetes.io/health-probe-timeout":                "10",
		"appgw.ingress.kubernetes.io/health-probe-unhealthy-threshold":    "3",
		"appgw.ingress.kubernetes.io/rewrite-rule-set":                    "my-rewrite-rule-set",
		"appgw.ingress.kubernetes.io/rewrite-rule-set-crd":                "my-rewrite-rule-set-crd",
		"kubernetes.io/ingress.class":                                     "azure/application-gateway",
		"appgw.ingress.istio.io/v1alpha3":                                 "azure/application-gateway",
		"falseKey":                                                        "false",
		"errorKey":                                                        "234error!!",
	}

	ing := &networking.Ingress{
		ObjectMeta: v1.ObjectMeta{
			Annotations: annotations,
		},
	}

	Context("test IsCookieBasedAffinity", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := IsCookieBasedAffinity(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(false))
		})
		It("returns true", func() {
			actual, err := IsCookieBasedAffinity(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(true))
		})
	})

	Context("test IsCookieBasedAffinityDistinctName", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := IsCookieBasedAffinityDistinctName(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(false))
		})
		It("returns true", func() {
			actual, err := IsCookieBasedAffinityDistinctName(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(true))
		})
	})

	Context("test appgwSslCertificate", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := GetAppGwSslCertificate(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(""))
		})
		It("returns true", func() {
			actual, err := GetAppGwSslCertificate(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal("appgw-cert"))
		})
	})

	Context("test appgwSslProfile", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := GetAppGwSslProfile(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(""))
		})
		It("returns true", func() {
			actual, err := GetAppGwSslProfile(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal("legacy-tls"))
		})
	})

	Context("test appgwTrustedRootCertificate", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := GetAppGwTrustedRootCertificate(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(""))
		})
		It("returns true", func() {
			actual, err := GetAppGwTrustedRootCertificate(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal("appgw-root-cert1,appgw-root-cert2"))
		})
	})

	Context("test health-probe-hostname", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := HealthProbeHostName(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(""))
		})
		It("returns hostname", func() {
			actual, err := HealthProbeHostName(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal("myhost.mydomain.com"))
		})
	})

	Context("test health-probe-port", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := HealthProbePort(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(int32(0)))
		})
		It("returns probe port", func() {
			actual, err := HealthProbePort(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(int32(8080)))
		})
	})

	Context("test health-probe-path", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := HealthProbePath(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(""))
		})
		It("returns probe path", func() {
			actual, err := HealthProbePath(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal("/healthz"))
		})
	})

	Context("test health-probe-status-codes", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := HealthProbeStatusCodes(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).Should(BeEmpty())
		})
		It("returns expected status codes", func() {
			actual, err := HealthProbeStatusCodes(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).Should(HaveLen(2))
			Expect(actual).Should(ConsistOf("200-399", "401"))
		})
	})

	Context("test health-probe-interval", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := HealthProbeInterval(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(int32(0)))
		})
		It("returns hostname", func() {
			actual, err := HealthProbeInterval(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(int32(15)))
		})
	})

	Context("test health-probe-timeout", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := HealthProbeTimeout(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(int32(0)))
		})
		It("returns hostname", func() {
			actual, err := HealthProbeTimeout(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(int32(10)))
		})
	})

	Context("test health-probe-unhealthy-threshold", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := HealthProbeUnhealthyThreshold(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(int32(0)))
		})
		It("returns hostname", func() {
			actual, err := HealthProbeUnhealthyThreshold(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(int32(3)))
		})
	})

	Context("test rewrite-rule-set", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := RewriteRuleSet(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(""))
		})
		It("returns rewrite rule set", func() {
			actual, err := RewriteRuleSet(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal("my-rewrite-rule-set"))
		})
	})

	Context("test rewrite-rule-set-crd", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := RewriteRuleSetCRD(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(""))
		})
		It("returns rewrite rule set", func() {
			actual, err := RewriteRuleSetCRD(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal("my-rewrite-rule-set-crd"))
		})
	})

	Context("test ConnectionDrainingTimeout", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := ConnectionDrainingTimeout(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(int32(0)))
		})
		It("returns the timeout", func() {
			actual, err := ConnectionDrainingTimeout(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(int32(3456)))
		})
	})

	Context("test IsConnectionDraining", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := IsConnectionDraining(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(false))
		})
		It("returns true", func() {
			actual, err := IsConnectionDraining(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(true))
		})
	})

	Context("test RequestTimeout", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := RequestTimeout(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(int32(0)))
		})
		It("returns the timeout", func() {
			actual, err := RequestTimeout(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(int32(123456)))
		})
	})

	Context("test BackendPathPrefix", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := BackendPathPrefix(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(""))
		})
		It("returns the prefix", func() {
			actual, err := BackendPathPrefix(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal("prefix-here"))
		})
	})

	Context("test BackendHostName", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := BackendHostName(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(""))
		})
		It("returns the hostname", func() {
			actual, err := BackendHostName(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal("www.backend.com"))
		})
	})

	Context("test IsSslRedirect", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := IsSslRedirect(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(false))
		})
		It("returns true with correct annotation", func() {
			actual, err := IsSslRedirect(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(true))
		})
	})

	Context("test UsePrivateIP", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, err := UsePrivateIP(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(false))
		})
		It("returns true with correct annotation", func() {
			actual, err := UsePrivateIP(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(true))
		})
	})

	Context("test OverrideFrontendPort", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			actual, _ := OverrideFrontendPort(ing)
			Expect(actual).To(Equal(int32(0)))
		})
		It("returns true with correct annotation", func() {
			actual, _ := OverrideFrontendPort(ing)
			Expect(actual).To(Equal(int32(444)))
		})
	})

	Context("test GetHostNameExtensions", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &networking.Ingress{}
			hostnames, err := GetHostNameExtensions(ing)
			Expect(err).To(HaveOccurred())
			Expect(hostnames).To(BeNil())
		})

		It("parses the hostname-extension correctly at correct delimiter", func() {
			ing := &networking.Ingress{}
			ing.Annotations = map[string]string{
				"appgw.ingress.kubernetes.io/hostname-extension": " www.bye.com ,  www.b*.com ",
			}
			hostnames, err := GetHostNameExtensions(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostnames).To(Equal([]string{"www.bye.com", "www.b*.com"}))
		})

		It("returns correct hostnames", func() {
			hostnames, err := GetHostNameExtensions(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(hostnames).To(Equal([]string{"www.bye.com", "www.b*.com"}))
		})
	})

	Context("test parseBol", func() {
		It("returns true", func() {
			actual, err := parseBool(ing, UsePrivateIPKey)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(true))
		})

		It("returns false", func() {
			actual, err := parseBool(ing, "falseKey")
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(false))
		})

		It("returns an error", func() {
			actual, err := parseBool(ing, "errorKey")
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(false))
		})
	})
})

func TestParseBoolTrue(t *testing.T) {
	key := "key"
	value := "true"
	ingress.Annotations[key] = value
	parsedVal, err := parseBool(&ingress, key)
	if !parsedVal || err != nil {
		t.Error(fmt.Sprintf(NoError, value, parsedVal, err))
	}
}

func TestParseBoolFalse(t *testing.T) {
	key := "key"
	value := "false"
	ingress.Annotations[key] = value
	parsedVal, err := parseBool(&ingress, key)
	if parsedVal || err != nil {
		t.Error(fmt.Sprintf(NoError, value, parsedVal, err))
	}
}

func TestParseBoolInvalid(t *testing.T) {
	key := "key"
	value := "nope"
	ingress.Annotations[key] = value
	parsedVal, err := parseBool(&ingress, key)
	if !controllererrors.IsErrorCode(err, controllererrors.ErrorInvalidContent) {
		t.Error(fmt.Sprintf(Error, controllererrors.ErrorInvalidContent, parsedVal, err))
	}
}

func TestParseBoolMissingKey(t *testing.T) {
	key := "key"
	delete(ingress.Annotations, key)
	parsedVal, err := parseBool(&ingress, key)
	if !controllererrors.IsErrorCode(err, controllererrors.ErrorMissingAnnotation) || parsedVal {
		t.Error(fmt.Sprintf(Error, controllererrors.ErrorMissingAnnotation, parsedVal, err))
	}
}

func TestParseInt32(t *testing.T) {
	key := "key"
	value := "20"
	ingress.Annotations[key] = value
	parsedVal, err := parseInt32(&ingress, key)
	if err != nil || fmt.Sprint(parsedVal) != value {
		t.Error(fmt.Sprintf(NoError, value, parsedVal, err))
	}
}

func TestParseInt32Invalid(t *testing.T) {
	key := "key"
	value := "20asd"
	ingress.Annotations[key] = value
	parsedVal, err := parseInt32(&ingress, key)
	if !controllererrors.IsErrorCode(err, controllererrors.ErrorInvalidContent) {
		t.Error(fmt.Sprintf(Error, controllererrors.ErrorInvalidContent, parsedVal, err))
	}
}

func TestParseInt32MissingKey(t *testing.T) {
	key := "key"
	delete(ingress.Annotations, key)
	parsedVal, err := parseInt32(&ingress, key)
	if !controllererrors.IsErrorCode(err, controllererrors.ErrorMissingAnnotation) || parsedVal != 0 {
		t.Error(fmt.Sprintf(Error, controllererrors.ErrorMissingAnnotation, parsedVal, err))
	}
}

func TestParseString(t *testing.T) {
	key := "key"
	value := "/path"
	ingress.Annotations[key] = value
	parsedVal, err := parseString(&ingress, key)
	if parsedVal != value || err != nil {
		t.Error(fmt.Sprintf(NoError, value, parsedVal, err))
	}
}

func TestParseStringMissingKey(t *testing.T) {
	key := "key"
	delete(ingress.Annotations, key)
	parsedVal, err := parseString(&ingress, key)
	if !controllererrors.IsErrorCode(err, controllererrors.ErrorMissingAnnotation) {
		t.Error(fmt.Sprintf(Error, controllererrors.ErrorMissingAnnotation, parsedVal, err))
	}
}
