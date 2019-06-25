// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"encoding/base64"
	"fmt"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"sort"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
)

// getSslCertificates obtains all SSL Certificates for the given Ingress object.
func (c *appGwConfigBuilder) getSslCertificates(ingressList []*v1beta1.Ingress) *[]network.ApplicationGatewaySslCertificate {
	secretIDCertificateMap := make(map[secretIdentifier]*string)

	for _, ingress := range ingressList {
		for k, v := range c.getSecretToCertificateMap(ingress) {
			secretIDCertificateMap[k] = v
		}
	}

	var sslCertificates []network.ApplicationGatewaySslCertificate
	for secretID, cert := range secretIDCertificateMap {
		sslCertificates = append(sslCertificates, newCert(secretID, cert))
	}
	sort.Sort(sorter.ByCertificateName(sslCertificates))
	return &sslCertificates
}

func (c *appGwConfigBuilder) getSecretToCertificateMap(ingress *v1beta1.Ingress) map[secretIdentifier]*string {
	secretIDCertificateMap := make(map[secretIdentifier]*string)
	for _, tls := range ingress.Spec.TLS {
		if len(tls.SecretName) == 0 {
			continue
		}

		tlsSecret := secretIdentifier{
			Name:      tls.SecretName,
			Namespace: ingress.Namespace,
		}

		// add hostname-tlsSecret mapping to a per-ingress map
		if cert := c.k8sContext.CertificateSecretStore.GetPfxCertificate(tlsSecret.secretKey()); cert != nil {
			secretIDCertificateMap[tlsSecret] = to.StringPtr(base64.StdEncoding.EncodeToString(cert))
		} else {
			logLine := fmt.Sprintf("Unable to find the secret associated to secretId: [%s]", tlsSecret.secretKey())
			c.recorder.Event(ingress, v1.EventTypeWarning, events.ReasonSecretNotFound, logLine)
		}
	}
	return secretIDCertificateMap
}

func (c *appGwConfigBuilder) getCertificate(ingress *v1beta1.Ingress, hostname string, hostnameSecretIDMap map[string]secretIdentifier) (*string, *secretIdentifier) {
	if hostnameSecretIDMap == nil {
		return nil, nil
	}
	secID, exists := hostnameSecretIDMap[hostname]
	if !exists {
		// check if wildcard exists
		secID, exists = hostnameSecretIDMap[""]
	}
	if !exists {
		// no wildcard or matched certificate
		return nil, nil
	}

	cert, exists := c.getSecretToCertificateMap(ingress)[secID]
	if !exists {
		// secret referred does not correspond to a certificate
		return nil, nil
	}
	return cert, &secID
}

func (c *appGwConfigBuilder) newHostToSecretMap(ingress *v1beta1.Ingress) map[string]secretIdentifier {
	hostToSecretMap := make(map[string]secretIdentifier)
	for _, tls := range ingress.Spec.TLS {
		if len(tls.SecretName) == 0 {
			continue
		}

		tlsSecret := secretIdentifier{
			Name:      tls.SecretName,
			Namespace: ingress.Namespace,
		}

		// add hostname-tlsSecret mapping to a per-ingress map
		cert := c.k8sContext.CertificateSecretStore.GetPfxCertificate(tlsSecret.secretKey())
		if cert == nil {
			continue
		}

		// default secret
		if len(tls.Hosts) == 0 {
			hostToSecretMap[""] = tlsSecret
		}

		for _, hostname := range tls.Hosts {
			// default secret
			if len(hostname) == 0 {
				hostToSecretMap[""] = tlsSecret
			} else {
				hostToSecretMap[hostname] = tlsSecret
			}
		}
	}
	return hostToSecretMap
}

func newCert(secretID secretIdentifier, cert *string) network.ApplicationGatewaySslCertificate {
	return network.ApplicationGatewaySslCertificate{
		Etag: to.StringPtr("*"),
		Name: to.StringPtr(secretID.secretFullName()),
		ApplicationGatewaySslCertificatePropertiesFormat: &network.ApplicationGatewaySslCertificatePropertiesFormat{
			Data:     cert,
			Password: to.StringPtr("msazure"),
		},
	}
}
