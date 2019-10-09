package controller

import (
	"fmt"
	"sync"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-06-01/network"
	"github.com/golang/glog"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/brownfield"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

type pruneFunc func(c *AppGwIngressController, appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, ingressList []*v1beta1.Ingress) []*v1beta1.Ingress

var once sync.Once
var pruneFuncList []pruneFunc

// PruneIngress filters ingress list based on filter functions and returns a filtered ingress list
func (c *AppGwIngressController) PruneIngress(appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext) []*v1beta1.Ingress {
	once.Do(func() {
		if cbCtx.EnvVariables.EnableBrownfieldDeployment {
			pruneFuncList = append(pruneFuncList, pruneProhibitedIngress)
		}
		pruneFuncList = append(pruneFuncList, pruneNoPrivateIP)
		pruneFuncList = append(pruneFuncList, pruneRedirectWithNoTLS)
	})
	prunedIngresses := cbCtx.IngressList
	for _, prune := range pruneFuncList {
		prunedIngresses = prune(c, appGw, cbCtx, prunedIngresses)
	}

	return prunedIngresses
}

// pruneProhibitedIngress filters rules that are specified by prohibited target CRD
func pruneProhibitedIngress(c *AppGwIngressController, appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, ingressList []*v1beta1.Ingress) []*v1beta1.Ingress {
	// Mutate the list of Ingresses by removing ones that AGIC should not be creating configuration.
	for idx, ingress := range ingressList {
		glog.V(5).Infof("Original Ingress[%d] Rules: %+v", idx, ingress.Spec.Rules)
		ingressList[idx].Spec.Rules = brownfield.PruneIngressRules(ingress, cbCtx.ProhibitedTargets)
		glog.V(5).Infof("Sanitized Ingress[%d] Rules: %+v", idx, ingress.Spec.Rules)
	}

	return ingressList
}

// pruneNoPrivateIP filters ingresses which use private IP annotation when AppGw doesn't have a private IP
func pruneNoPrivateIP(c *AppGwIngressController, appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, ingressList []*v1beta1.Ingress) []*v1beta1.Ingress {
	var prunedIngresses []*v1beta1.Ingress
	appGwHasPrivateIP := appgw.LookupIPConfigurationByType(appGw.FrontendIPConfigurations, true) != nil
	for _, ingress := range ingressList {
		usePrivateIP, err := annotations.UsePrivateIP(ingress)
		if err != nil && annotations.IsInvalidContent(err) {
			glog.Errorf("Ingress %s/%s has invalid value for annotation %s", ingress.Namespace, ingress.Name, annotations.UsePrivateIPKey)
		}

		usePrivateIP = usePrivateIP || cbCtx.EnvVariables.UsePrivateIP == "true"
		if usePrivateIP && !appGwHasPrivateIP {
			errorLine := fmt.Sprintf("ignoring Ingress %s/%s as it requires Application Gateway %s has a private IP adress", ingress.Namespace, ingress.Name, c.appGwIdentifier.AppGwName)
			glog.Error(errorLine)
			c.recorder.Event(ingress, v1.EventTypeWarning, events.ReasonNoPrivateIPError, errorLine)
			c.logToAGICEvents(v1.EventTypeWarning, events.ReasonNoPrivateIPError, errorLine)
		} else {
			prunedIngresses = append(prunedIngresses, ingress)
		}
	}

	return prunedIngresses
}

// pruneRedirectWithNoTLS filters ingresses which are annotated for ssl redirect but don't have a TLS section in the spec
func pruneRedirectWithNoTLS(c *AppGwIngressController, appGw *n.ApplicationGateway, cbCtx *appgw.ConfigBuilderContext, ingressList []*v1beta1.Ingress) []*v1beta1.Ingress {
	var prunedIngresses []*v1beta1.Ingress
	for _, ingress := range ingressList {
		hasTLS := ingress.Spec.TLS != nil && len(ingress.Spec.TLS) > 0
		sslRedirect, _ := annotations.IsSslRedirect(ingress)
		if !hasTLS && sslRedirect {
			errorLine := fmt.Sprintf("ignoring Ingress %s/%s as it has an invalid spec. It is annotated with ssl-redirect: true but is missing a TLS secret. Please add a TLS secret or remove ssl-redirect annotation", ingress.Namespace, ingress.Name)
			glog.Error(errorLine)
			c.recorder.Event(ingress, v1.EventTypeWarning, events.ReasonRedirectWithNoTLS, errorLine)
			c.logToAGICEvents(v1.EventTypeWarning, events.ReasonRedirectWithNoTLS, errorLine)
		} else {
			prunedIngresses = append(prunedIngresses, ingress)
		}
	}

	return prunedIngresses
}
