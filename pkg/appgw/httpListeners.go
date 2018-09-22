// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"encoding/base64"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"k8s.io/api/extensions/v1beta1"
)

func (builder *appGwConfigBuilder) getCertificate(ingressKey string, hostname string) (*string, *secretIdentifier) {
	_, exists := builder.ingressKeyHostnameSecretIDMap[ingressKey]
	if !exists {
		return nil, nil
	}
	secID, exists := builder.ingressKeyHostnameSecretIDMap[ingressKey][hostname]
	if !exists {
		// check if wildcard exists
		secID, exists = builder.ingressKeyHostnameSecretIDMap[ingressKey][""]
	}
	if !exists {
		// no wildcard or matched certificate
		return nil, nil
	}
	cert, exists := builder.secretIDCertificateMap[secID]
	if !exists {
		// secret referred does not correspond to a certificate
		return nil, nil
	}
	return cert, &secID
}

func (builder *appGwConfigBuilder) HTTPListeners(ingressList [](*v1beta1.Ingress)) (ConfigBuilder, error) {
	frontendListeners := utils.NewUnorderedSet()
	frontendPortsSet := utils.NewUnorderedSet()
	builder.secretIDCertificateMap = make(map[secretIdentifier]*string)
	builder.ingressKeyHostnameSecretIDMap = make(map[string](map[string]secretIdentifier))
	for _, ingress := range ingressList {
		hostnameSecretIDMap := make(map[string](secretIdentifier))
		if len(ingress.Spec.TLS) > 0 {
			for _, tls := range ingress.Spec.TLS {
				if len(tls.SecretName) == 0 {
					continue
				}
				secretID := secretIdentifier{
					Name:      tls.SecretName,
					Namespace: ingress.Namespace,
				}

				// add hostname-secretID mapping to a per-ingress map
				cert := builder.k8sContext.CertificateSecretStore.GetPfxCertificate(secretID.secretKey())
				if cert == nil {
					continue
				}
				certEncoded := base64.StdEncoding.EncodeToString(cert)

				builder.secretIDCertificateMap[secretID] = &certEncoded

				// default secret
				if len(tls.Hosts) == 0 {
					hostnameSecretIDMap[""] = secretID
				}

				for _, hostname := range tls.Hosts {
					// default secret
					if len(hostname) == 0 {
						hostnameSecretIDMap[""] = secretID
					} else {
						hostnameSecretIDMap[hostname] = secretID
					}
				}
			}
		}
		ingressKey := getResourceKey(ingress.Namespace, ingress.Name)
		builder.ingressKeyHostnameSecretIDMap[ingressKey] = hostnameSecretIDMap

		for _, rule := range ingress.Spec.Rules {
			if rule.HTTP == nil {
				// skip no http rule
				continue
			}

			httpsAvailable := false
			cert, secID := builder.getCertificate(ingressKey, rule.Host)
			if cert != nil {
				httpsAvailable = true
			}

			// TODO skip this part if HTTP is disabled in annotation
			listenerConfigHTTP := frontendListenerAzureConfig{
				Protocol: network.HTTP,
			}
			listenerIDHTTP := generateFrontendListenerID(&rule, listenerConfigHTTP.Protocol, nil)
			frontendListeners.Insert(listenerIDHTTP)
			frontendPortsSet.Insert(listenerIDHTTP.FrontendPort)
			builder.httpListenersAzureConfigMap[listenerIDHTTP] = &listenerConfigHTTP

			// HTTPS is also available
			if httpsAvailable {
				listenerConfigHTTPS := frontendListenerAzureConfig{
					Protocol: network.HTTPS,
					Secret:   *secID,
				}
				listenerIDHTTPS := generateFrontendListenerID(&rule, listenerConfigHTTPS.Protocol, nil)
				frontendListeners.Insert(listenerIDHTTPS)
				frontendPortsSet.Insert(listenerIDHTTPS.FrontendPort)
				builder.httpListenersAzureConfigMap[listenerIDHTTPS] = &listenerConfigHTTPS
			}
		}
	}

	sslCertificates := []network.ApplicationGatewaySslCertificate{}

	// add all the certificates
	for secretID, cert := range builder.secretIDCertificateMap {
		sslCertificateName := secretID.secretFullName()
		sslCertificates = append(sslCertificates, network.ApplicationGatewaySslCertificate{
			Etag: to.StringPtr("*"),
			Name: &sslCertificateName,
			ApplicationGatewaySslCertificatePropertiesFormat: &network.ApplicationGatewaySslCertificatePropertiesFormat{
				Data:     cert,
				Password: to.StringPtr("msazure"),
			},
		})
	}

	httpListeners := []network.ApplicationGatewayHTTPListener{}
	frontendPorts := []network.ApplicationGatewayFrontendPort{}

	// fallback to default listener as placeholder if no listener is available
	if frontendPortsSet.IsEmpty() {
		d := defaultFrontendListenerIdentifier()
		frontendPortsSet.Insert(d.FrontendPort)
		frontendListeners.Insert(d)
	}

	frontendPortsSet.ForEach(func(frontendPortInterface interface{}) {
		frontendPort := frontendPortInterface.(int32)
		frontendPortName := generateFrontendPortName(frontendPort)
		frontendPorts = append(frontendPorts, network.ApplicationGatewayFrontendPort{
			Etag: to.StringPtr("*"),
			Name: &frontendPortName,
			ApplicationGatewayFrontendPortPropertiesFormat: &network.ApplicationGatewayFrontendPortPropertiesFormat{
				Port: to.Int32Ptr(frontendPort),
			},
		})
	})

	frontendListeners.ForEach(func(frontendListenerIDInterface interface{}) {
		frontendListenerID := frontendListenerIDInterface.(frontendListenerIdentifier)
		frontendListenerConfig := builder.httpListenersAzureConfigMap[frontendListenerID]
		if frontendListenerConfig == nil {
			// use default
			frontendListenerConfig = &frontendListenerAzureConfig{
				Protocol: network.HTTP,
			}
		}

		frontendPort := frontendListenerID.FrontendPort
		frontendPortName := generateFrontendPortName(frontendPort)
		frontendPortID := builder.appGwIdentifier.frontendPortID(frontendPortName)

		httpListenerName := generateHTTPListenerName(frontendListenerID)
		httpListener := network.ApplicationGatewayHTTPListener{
			Etag: to.StringPtr("*"),
			Name: &httpListenerName,
			ApplicationGatewayHTTPListenerPropertiesFormat: &network.ApplicationGatewayHTTPListenerPropertiesFormat{
				// TODO: expose this to external configuration
				FrontendIPConfiguration: resourceRef(*(*builder.appGwConfig.FrontendIPConfigurations)[0].ID),
				FrontendPort:            resourceRef(frontendPortID),
				Protocol:                frontendListenerConfig.Protocol,
				HostName:                &frontendListenerID.HostName,
			},
		}

		if frontendListenerConfig.Protocol == network.HTTPS {
			sslCertificateName := frontendListenerConfig.Secret.secretFullName()
			sslCertificateID := builder.appGwIdentifier.sslCertificateID(sslCertificateName)

			httpListener.SslCertificate = resourceRef(sslCertificateID)

			if len(*httpListener.ApplicationGatewayHTTPListenerPropertiesFormat.HostName) != 0 {
				httpListener.RequireServerNameIndication = to.BoolPtr(true)
			}
		}

		if len(*httpListener.ApplicationGatewayHTTPListenerPropertiesFormat.HostName) != 0 {
			httpListeners = append([]network.ApplicationGatewayHTTPListener{httpListener}, httpListeners...)
		} else {
			httpListeners = append(httpListeners, httpListener)
		}

		builder.httpListenersMap[frontendListenerID] = &httpListener
	})

	builder.appGwConfig.SslCertificates = &sslCertificates
	builder.appGwConfig.FrontendPorts = &frontendPorts
	builder.appGwConfig.HTTPListeners = &httpListeners
	return builder, nil
}
