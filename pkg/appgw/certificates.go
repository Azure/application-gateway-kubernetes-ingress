// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"encoding/base64"
	"fmt"
	"sort"
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/sorter"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
)

// getSslCertificates obtains all SSL Certificates for the given Ingress object.
func (c *appGwConfigBuilder) getSslCertificates(cbCtx *ConfigBuilderContext) *[]n.ApplicationGatewaySslCertificate {
	if c.mem.certs != nil {
		return c.mem.certs
	}
	secretIDCertificateMap := make(map[secretIdentifier]*string)

	for _, ingress := range cbCtx.IngressList {
		for k, v := range c.getSecretToCertificateMap(ingress) {
			secretIDCertificateMap[k] = v
		}
	}

	sslCertificates := []n.ApplicationGatewaySslCertificate{}
	for secretID, cert := range secretIDCertificateMap {
		sslCertificates = append(sslCertificates, c.newCert(secretID, cert))
	}

	desiredCertNames := make(map[string]struct{}, len(sslCertificates))
	for _, cert := range sslCertificates {
		if cert.Name == nil {
			continue
		}
		desiredCertNames[*cert.Name] = struct{}{}
	}

	// Certs referenced by the appgw-ssl-certificate annotation point to already-installed certificates on the gateway.
	// These certs must never be garbage-collected, even if their name matches the AGIC-managed naming pattern.
	annotatedCertNames := make(map[string]struct{})
	for _, ingress := range cbCtx.IngressList {
		certName, _ := annotations.GetAppGwSslCertificate(ingress)
		if certName == "" {
			continue
		}
		annotatedCertNames[certName] = struct{}{}
	}

	// Retain only non-AGIC/manual certificates from the existing gateway config.
	// In brownfield mode, also retain any certificates referenced by blacklisted listeners.
	if c.appGw.SslCertificates != nil {
		referencedByBlacklisted := c.sslCertNamesReferencedByBlacklistedListeners(cbCtx)
		retainedExisting := make([]n.ApplicationGatewaySslCertificate, 0, len(*c.appGw.SslCertificates))
		for _, existingCert := range *c.appGw.SslCertificates {
			if existingCert.Name == nil {
				continue
			}
			existingName := *existingCert.Name
			if _, required := annotatedCertNames[existingName]; required {
				retainedExisting = append(retainedExisting, existingCert)
				continue
			}
			if _, desired := desiredCertNames[existingName]; desired {
				// The desired cert will be included (and will overwrite by name if needed).
				continue
			}

			if !isAgicManagedSslCertificateName(existingName) {
				retainedExisting = append(retainedExisting, existingCert)
				continue
			}

			if cbCtx.EnvVariables.EnableBrownfieldDeployment {
				if _, keep := referencedByBlacklisted[existingName]; keep {
					retainedExisting = append(retainedExisting, existingCert)
				}
			}
		}

		sslCertificates = brownfield.MergeCerts(retainedExisting, sslCertificates)
	}

	sort.Sort(sorter.ByCertificateName(sslCertificates))
	c.mem.certs = &sslCertificates
	return &sslCertificates
}

func isAgicManagedSslCertificateName(name string) bool {
	// AGIC-generated certificate names follow secretIdentifier.secretFullName():
	//   <APPGW_CONFIG_NAME_PREFIX>cert-<namespace>-<secretName>
	// Preinstalled/manual certificates (including those referenced via appgw-ssl-certificate annotation)
	// do not follow this pattern and should be retained.
	return strings.HasPrefix(name, agPrefix+prefixSslCertificate+"-")
}

func (c *appGwConfigBuilder) sslCertNamesReferencedByBlacklistedListeners(cbCtx *ConfigBuilderContext) map[string]struct{} {
	result := make(map[string]struct{})
	if !cbCtx.EnvVariables.EnableBrownfieldDeployment {
		return result
	}

	er := brownfield.NewExistingResources(c.appGw, cbCtx.ProhibitedTargets, nil)
	blacklistedListeners, _ := er.GetBlacklistedListeners()
	for _, listener := range blacklistedListeners {
		if listener.SslCertificate == nil || listener.SslCertificate.ID == nil {
			continue
		}
		certName := utils.GetLastChunkOfSlashed(*listener.SslCertificate.ID)
		if certName == "" {
			continue
		}
		result[certName] = struct{}{}
	}
	return result
}

func (c *appGwConfigBuilder) getSecretToCertificateMap(ingress *networking.Ingress) map[secretIdentifier]*string {
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

func (c *appGwConfigBuilder) getCertificate(ingress *networking.Ingress, hostname string, hostnameSecretIDMap map[string]secretIdentifier) (*string, *secretIdentifier) {
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

func (c *appGwConfigBuilder) newHostToSecretMap(ingress *networking.Ingress) map[string]secretIdentifier {
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

func (c *appGwConfigBuilder) newCert(secretID secretIdentifier, cert *string) n.ApplicationGatewaySslCertificate {
	sslCertName := secretID.secretFullName()
	return n.ApplicationGatewaySslCertificate{
		Etag: to.StringPtr("*"),
		Name: to.StringPtr(sslCertName),
		ID:   to.StringPtr(c.appGwIdentifier.sslCertificateID(sslCertName)),
		ApplicationGatewaySslCertificatePropertiesFormat: &n.ApplicationGatewaySslCertificatePropertiesFormat{
			Data:     cert,
			Password: to.StringPtr("msazure"),
		},
	}
}
