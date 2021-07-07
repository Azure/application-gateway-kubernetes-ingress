package controller

import (
	"fmt"
	"strings"
	"sync"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

type pruneFunc func(c *AppGwIngressController, appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, ingressList []*networking.Ingress) []*networking.Ingress

var once sync.Once
var pruneFuncList []pruneFunc

// PruneIngress filters ingress list based on filter functions and returns a filtered ingress list
func (c *AppGwIngressController) PruneIngress(appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext) []*networking.Ingress {
	once.Do(func() {
		if cbCtx.EnvVariables.EnableBrownfieldDeployment {
			pruneFuncList = append(pruneFuncList, pruneProhibitedIngress)
		}
		pruneFuncList = append(pruneFuncList, pruneNoPrivateIP)
		pruneFuncList = append(pruneFuncList, pruneRedirectWithNoTLS)
		pruneFuncList = append(pruneFuncList, pruneNoSslCertificate)
		pruneFuncList = append(pruneFuncList, pruneNoTrustedRootCertificate)
	})
	prunedIngresses := cbCtx.IngressList
	for _, prune := range pruneFuncList {
		prunedIngresses = prune(c, appGw, cbCtx, prunedIngresses)
	}

	return prunedIngresses
}

// pruneProhibitedIngress filters rules that are specified by prohibited target CRD
func pruneProhibitedIngress(c *AppGwIngressController, appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, ingressList []*networking.Ingress) []*networking.Ingress {
	// Mutate the list of Ingresses by removing ones that AGIC should not be creating configuration.
	for idx, ingress := range ingressList {
		klog.V(5).Infof("Original Ingress[%d] Rules: %+v", idx, ingress.Spec.Rules)
		ingressList[idx].Spec.Rules = brownfield.PruneIngressRules(ingress, cbCtx.ProhibitedTargets)
		klog.V(5).Infof("Sanitized Ingress[%d] Rules: %+v", idx, ingress.Spec.Rules)
	}

	return ingressList
}

// pruneNoPrivateIP filters ingresses which use private IP annotation when AppGw doesn't have a private IP
func pruneNoPrivateIP(c *AppGwIngressController, appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, ingressList []*networking.Ingress) []*networking.Ingress {
	var prunedIngresses []*networking.Ingress
	appGwHasPrivateIP := appgw.LookupIPConfigurationByType(appGw.FrontendIPConfigurations, true) != nil
	for _, ingress := range ingressList {
		usePrivateIP, err := annotations.UsePrivateIP(ingress)
		if err != nil && controllererrors.IsErrorCode(err, controllererrors.ErrorInvalidContent) {
			klog.Errorf("Ingress %s/%s has invalid value for annotation %s", ingress.Namespace, ingress.Name, annotations.UsePrivateIPKey)
		}

		usePrivateIP = usePrivateIP || cbCtx.EnvVariables.UsePrivateIP
		if usePrivateIP && !appGwHasPrivateIP {
			errorLine := fmt.Sprintf("ignoring Ingress %s/%s as it requires Application Gateway %s has a private IP adress", ingress.Namespace, ingress.Name, c.appGwIdentifier.AppGwName)
			klog.Error(errorLine)
			c.recorder.Event(ingress, v1.EventTypeWarning, events.ReasonNoPrivateIPError, errorLine)
			if c.agicPod != nil {
				c.recorder.Event(c.agicPod, v1.EventTypeWarning, events.ReasonNoPrivateIPError, errorLine)
			}
		} else {
			prunedIngresses = append(prunedIngresses, ingress)
		}
	}

	return prunedIngresses
}

// pruneNoSslCertificate filters ingresses which use appgw-ssl-certificate annotation when AppGw doesn't have annotated ssl certificate installed
func pruneNoSslCertificate(c *AppGwIngressController, appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, ingressList []*networking.Ingress) []*networking.Ingress {
	var prunedIngresses []*networking.Ingress
	set := make(map[string]bool)
	for _, installedSslCertificate := range *appGw.SslCertificates {
		set[*installedSslCertificate.Name] = true
	}

	for _, ingress := range ingressList {
		annotatedSslCertificate, err := annotations.GetAppGwSslCertificate(ingress)
		// if annotation is not specified, add the ingress and go check next
		if err != nil && controllererrors.IsErrorCode(err, controllererrors.ErrorMissingAnnotation) {
			prunedIngresses = append(prunedIngresses, ingress)
			continue
		}

		// given empty string is a valid annotation value, we error out with a message if no match
		if _, exists := set[annotatedSslCertificate]; !exists {
			errorLine := fmt.Sprintf("ignoring Ingress %s/%s as it requires Application Gateway %s to have pre-installed ssl certificate '%s'", ingress.Namespace, ingress.Name, c.appGwIdentifier.AppGwName, annotatedSslCertificate)
			klog.Error(errorLine)
			c.recorder.Event(ingress, v1.EventTypeWarning, events.ReasonNoPreInstalledSslCertificate, errorLine)
			if c.agicPod != nil {
				c.recorder.Event(c.agicPod, v1.EventTypeWarning, events.ReasonNoPreInstalledSslCertificate, errorLine)
			}
		} else {
			prunedIngresses = append(prunedIngresses, ingress)
		}
	}

	return prunedIngresses
}

// pruneNoTrustedRootCertificate filters ingresses which use appgw-trusted-root-certificate annotation when AppGw doesn't have annotated root certificate(s) installed
func pruneNoTrustedRootCertificate(c *AppGwIngressController, appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, ingressList []*networking.Ingress) []*networking.Ingress {
	var prunedIngresses []*networking.Ingress
	set := make(map[string]bool)
	for _, installedTrustedRootCertificate := range *appGw.TrustedRootCertificates {
		set[*installedTrustedRootCertificate.Name] = true
	}

	for _, ingress := range ingressList {
		installed := true
		trustedRootCertificates, err := annotations.GetAppGwTrustedRootCertificate(ingress)
		// if annotation is not specified
		if err != nil && controllererrors.IsErrorCode(err, controllererrors.ErrorMissingAnnotation) {
			prunedIngresses = append(prunedIngresses, ingress)
			continue
		}

		for _, rootCert := range strings.Split(trustedRootCertificates, ",") {
			if _, exists := set[rootCert]; !exists {
				installed = false
				errorLine := fmt.Sprintf("ignoring Ingress %s/%s as it requires Application Gateway %s to have pre-installed root certificate '%s'", ingress.Namespace, ingress.Name, c.appGwIdentifier.AppGwName, rootCert)
				klog.Error(errorLine)
				c.recorder.Event(ingress, v1.EventTypeWarning, events.ReasonNoPreInstalledRootCertificate, errorLine)
				if c.agicPod != nil {
					c.recorder.Event(c.agicPod, v1.EventTypeWarning, events.ReasonNoPreInstalledRootCertificate, errorLine)
				}
			}
		}
		if installed {
			prunedIngresses = append(prunedIngresses, ingress)
		}
	}

	return prunedIngresses
}

// pruneRedirectWithNoTLS filters ingresses which are annotated for ssl redirect but don't have a TLS section in the spec
func pruneRedirectWithNoTLS(c *AppGwIngressController, appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, ingressList []*networking.Ingress) []*networking.Ingress {
	var prunedIngresses []*networking.Ingress
	for _, ingress := range ingressList {
		appgwCertName, _ := annotations.GetAppGwSslCertificate(ingress)
		hasTLS := (ingress.Spec.TLS != nil && len(ingress.Spec.TLS) > 0) || len(appgwCertName) > 0
		sslRedirect, _ := annotations.IsSslRedirect(ingress)
		if !hasTLS && sslRedirect {
			errorLine := fmt.Sprintf("ignoring Ingress %s/%s as it has an invalid spec. It is annotated with ssl-redirect: true but is missing a TLS secret or '%s' annotation. Please add a TLS secret/annotation or remove ssl-redirect annotation", ingress.Namespace, ingress.Name, annotations.AppGwSslCertificate)
			klog.Error(errorLine)
			c.recorder.Event(ingress, v1.EventTypeWarning, events.ReasonRedirectWithNoTLS, errorLine)
			if c.agicPod != nil {
				c.recorder.Event(c.agicPod, v1.EventTypeWarning, events.ReasonRedirectWithNoTLS, errorLine)
			}
		} else {
			prunedIngresses = append(prunedIngresses, ingress)
		}
	}

	return prunedIngresses
}
