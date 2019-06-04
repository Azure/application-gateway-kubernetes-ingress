package appgw

import (
	"strings"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"k8s.io/api/extensions/v1beta1"
)

// getListeners constructs the unique set of App Gateway HTTP listeners across all ingresses.
func (c *appGwConfigBuilder) getListeners(ingressList []*v1beta1.Ingress) (*[]n.ApplicationGatewayHTTPListener, map[listenerIdentifier]*n.ApplicationGatewayHTTPListener) {
	// TODO(draychev): this is for compatibility w/ RequestRoutingRules and should be removed ASAP
	legacyMap := make(map[listenerIdentifier]*n.ApplicationGatewayHTTPListener)

	var listeners []n.ApplicationGatewayHTTPListener

	for listenerID, config := range c.getListenerConfigs(ingressList) {
		listener := c.newListener(listenerID, config.Protocol)
		if config.Protocol == n.HTTPS {
			sslCertificateID := c.appGwIdentifier.sslCertificateID(config.Secret.secretFullName())
			listener.SslCertificate = resourceRef(sslCertificateID)
		}
		listeners = append(listeners, listener)
		legacyMap[listenerID] = &listener
	}

	// TODO(draychev): The second map we return is for compatibility w/ RequestRoutingRules and should be removed ASAP
	return &listeners, legacyMap
}

// getListenerConfigs creates an intermediary representation of the listener configs based on the passed list of ingresses
func (c *appGwConfigBuilder) getListenerConfigs(ingressList []*v1beta1.Ingress) map[listenerIdentifier]listenerAzConfig {
	allListeners := make(map[listenerIdentifier]listenerAzConfig)
	for _, ingress := range ingressList {
		_, azListenerConfigs := c.processIngressRules(ingress)
		for listenerID, azConfig := range azListenerConfigs {
			allListeners[listenerID] = azConfig
		}
	}

	// App Gateway must have at least one listener - the default one!
	if len(allListeners) == 0 {
		allListeners[defaultFrontendListenerIdentifier()] = listenerAzConfig{
			// Default protocol
			Protocol: n.HTTP,
		}
	}

	return allListeners
}

func (c *appGwConfigBuilder) newListener(listener listenerIdentifier, protocol n.ApplicationGatewayProtocol) n.ApplicationGatewayHTTPListener {
	frontendPortName := generateFrontendPortName(listener.FrontendPort)
	frontendPortID := c.appGwIdentifier.frontendPortID(frontendPortName)

	return n.ApplicationGatewayHTTPListener{
		Etag: to.StringPtr("*"),
		Name: to.StringPtr(generateListenerName(listener)),
		ApplicationGatewayHTTPListenerPropertiesFormat: &n.ApplicationGatewayHTTPListenerPropertiesFormat{
			// TODO: expose this to external configuration
			FrontendIPConfiguration: resourceRef(*c.getPublicIPID()),
			FrontendPort:            resourceRef(frontendPortID),
			Protocol:                protocol,
			HostName:                &listener.HostName,
		},
	}
}

func (c *appGwConfigBuilder) getPublicIPID() *string {
	var publicIPID *string
	jsonConfigs := make([]string, 0)
	for _, ip := range *c.appGwConfig.FrontendIPConfigurations {
		// Collect the JSON IP configs for debug purposes.
		if jsonConf, err := ip.MarshalJSON(); err != nil {
			glog.Error("Could not marshall IP configuration:", *ip.ID, err)
		} else {
			jsonConfigs = append(jsonConfigs, string(jsonConf))
		}
		// Either PublicIPAddress is nil or PrivateIPAddress; never both present never both nil;
		if ip.ApplicationGatewayFrontendIPConfigurationPropertiesFormat != nil && ip.PublicIPAddress != nil {
			publicIPID = ip.ID
		}
	}

	if publicIPID == nil {
		// App Gateway will always have a Public IP address.
		// In the case where somehow it does not have one - it may be appropriate to crash.
		ips := strings.Join(jsonConfigs, ", ")

		// Will call os.Exit(255)
		// TODO(draychev): glog.Fatal does not expose stack trace.
		glog.Fatal("HTTP Listener was not able to find a Public IP address for App Gateway. Available IPs:", ips)
	}

	return publicIPID
}
