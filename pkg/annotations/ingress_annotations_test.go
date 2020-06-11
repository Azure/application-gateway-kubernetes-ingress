// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

// +build unittest

package annotations

import (
	"fmt"
	"testing"

	"github.com/knative/pkg/apis/istio/v1alpha3"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/api/extensions/v1beta1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
)

var ingress = v1beta1.Ingress{
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
		"appgw.ingress.kubernetes.io/use-private-ip":                 "true",
		"appgw.ingress.kubernetes.io/connection-draining":            "true",
		"appgw.ingress.kubernetes.io/cookie-based-affinity":          "true",
		"appgw.ingress.kubernetes.io/ssl-redirect":                   "true",
		"appgw.ingress.kubernetes.io/request-timeout":                "123456",
		"appgw.ingress.kubernetes.io/connection-draining-timeout":    "3456",
		"appgw.ingress.kubernetes.io/backend-path-prefix":            "prefix-here",
		"appgw.ingress.kubernetes.io/backend-hostname":               "www.backend.com",
		"appgw.ingress.kubernetes.io/hostname-extension":             "www.bye.com, www.b*.com",
		"appgw.ingress.kubernetes.io/appgw-ssl-certificate":          "appgw-cert",
		"appgw.ingress.kubernetes.io/appgw-trusted-root-certificate": "appgw-root-cert1,appgw-root-cert2",
		"kubernetes.io/ingress.class":                                "azure/application-gateway",
		"appgw.ingress.istio.io/v1alpha3":                            "azure/application-gateway",
		"falseKey":                                                   "false",
		"errorKey":                                                   "234error!!",
	}

	ing := &v1beta1.Ingress{
		ObjectMeta: v1.ObjectMeta{
			Annotations: annotations,
		},
	}

	Context("test IsCookieBasedAffinity", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &v1beta1.Ingress{}
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

	Context("test appgwSslCertificate", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &v1beta1.Ingress{}
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

	Context("test appgwTrustedRootCertificate", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &v1beta1.Ingress{}
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

	Context("test ConnectionDrainingTimeout", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &v1beta1.Ingress{}
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
			ing := &v1beta1.Ingress{}
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
			ing := &v1beta1.Ingress{}
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
			ing := &v1beta1.Ingress{}
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
			ing := &v1beta1.Ingress{}
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
			ing := &v1beta1.Ingress{}
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

	Context("test IsIstioGatewayIngress", func() {
		It("returns error when gateway has no annotations", func() {
			gateway := &v1alpha3.Gateway{}
			actual, err := IsIstioGatewayIngress(gateway)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(false))
		})
		It("returns true with correct annotation", func() {
			gateway := &v1alpha3.Gateway{
				ObjectMeta: v1.ObjectMeta{
					Annotations: annotations,
				},
			}
			actual, err := IsIstioGatewayIngress(gateway)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(true))
		})
	})

	Context("test IsApplicationGatewayIngress", func() {

		BeforeEach(func() {
			ApplicationGatewayIngressClass = DefaultIngressClass
		})

		It("returns error when ingress has no annotations", func() {
			ing := &v1beta1.Ingress{}
			actual, err := IsApplicationGatewayIngress(ing)
			Expect(err).To(HaveOccurred())
			Expect(actual).To(Equal(false))
		})
		It("returns true with correct annotation", func() {
			actual, err := IsApplicationGatewayIngress(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(true))
		})

		It("returns true with correct annotation", func() {
			ing.Annotations[IngressClassKey] = "custom-class"
			ApplicationGatewayIngressClass = "custom-class"
			actual, err := IsApplicationGatewayIngress(ing)
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(true))
		})

		It("returns false with incorrect annotation", func() {
			ing.Annotations[IngressClassKey] = "custom-class"
			actual, err := IsApplicationGatewayIngress(ing)
			Expect(ApplicationGatewayIngressClass).To(Equal(DefaultIngressClass))
			Expect(err).ToNot(HaveOccurred())
			Expect(actual).To(Equal(false))
		})
	})

	Context("test UsePrivateIP", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &v1beta1.Ingress{}
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

	Context("test GetHostNameExtensions", func() {
		It("returns error when ingress has no annotations", func() {
			ing := &v1beta1.Ingress{}
			hostnames, err := GetHostNameExtensions(ing)
			Expect(err).To(HaveOccurred())
			Expect(hostnames).To(BeNil())
		})

		It("parses the hostname-extension correctly at correct delimiter", func() {
			ing := &v1beta1.Ingress{}
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
