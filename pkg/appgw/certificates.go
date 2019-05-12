package appgw

import (
	"encoding/base64"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/api/extensions/v1beta1"
)

func (builder *appGwConfigBuilder) getSslCertificates(ingressList []*v1beta1.Ingress) *[]network.ApplicationGatewaySslCertificate {
	secretIDCertificateMap := make(map[secretIdentifier]*string)

	for _, ingress := range ingressList {
		for k, v := range builder.getSecretToCertificateMap(ingress) {
			secretIDCertificateMap[k] = v
		}
	}

	var sslCertificates []network.ApplicationGatewaySslCertificate
	for secretID, cert := range secretIDCertificateMap {
		sslCertificates = append(sslCertificates, makeCert(secretID, cert))
	}
	return &sslCertificates
}

func (builder *appGwConfigBuilder) getSecretToCertificateMap(ingress *v1beta1.Ingress) map[secretIdentifier]*string {
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
		cert := builder.k8sContext.CertificateSecretStore.GetPfxCertificate(tlsSecret.secretKey())
		if cert == nil {
			continue
		}
		secretIDCertificateMap[tlsSecret] = to.StringPtr(base64.StdEncoding.EncodeToString(cert))
	}
	return secretIDCertificateMap
}

// TODO(draychev): Remove V1 when the old declaration of getCertificate(...) is removed from the code base
func (builder *appGwConfigBuilder) getCertificateV1(ingress *v1beta1.Ingress, hostname string, hostnameSecretIDMap map[string]secretIdentifier) (*string, *secretIdentifier) {
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

	cert, exists := builder.getSecretToCertificateMap(ingress)[secID]
	if !exists {
		// secret referred does not correspond to a certificate
		return nil, nil
	}
	return cert, &secID
}

func (builder *appGwConfigBuilder) makeHostToSecretMap(ingress *v1beta1.Ingress) map[string]secretIdentifier {
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
		cert := builder.k8sContext.CertificateSecretStore.GetPfxCertificate(tlsSecret.secretKey())
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

func makeCert(secretID secretIdentifier, cert *string) network.ApplicationGatewaySslCertificate {
	return network.ApplicationGatewaySslCertificate{
		Etag: to.StringPtr("*"),
		Name: to.StringPtr(secretID.secretFullName()),
		ApplicationGatewaySslCertificatePropertiesFormat: &network.ApplicationGatewaySslCertificatePropertiesFormat{
			Data:     cert,
			Password: to.StringPtr("msazure"),
		},
	}
}
