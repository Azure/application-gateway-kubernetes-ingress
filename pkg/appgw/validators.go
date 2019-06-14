// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"errors"
	"fmt"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/client-go/tools/record"
)

const (
	errKeyNoDefaults     = "no-defaults"
	errKeyEitherDefaults = "either-defaults"
	errKeyNoBorR         = "no-backend-or-redirect"
	errKeyEitherBorR     = "either-backend-or-redirect"
)

var validationErrors = map[string]error{
	errKeyNoDefaults:     errors.New("either a DefaultRedirectConfiguration or (DefaultBackendAddressPool + DefaultBackendHTTPSettings) must be configured"),
	errKeyEitherDefaults: errors.New("URL Path Map must have either DefaultRedirectConfiguration or (DefaultBackendAddressPool + DefaultBackendHTTPSettings) but not both"),
	errKeyNoBorR:         errors.New("A valid path rule must have one of RedirectConfiguration or (BackendAddressPool + BackendHTTPSettings)"),
	errKeyEitherBorR:     errors.New("A Path Rule must have either RedirectConfiguration or (BackendAddressPool + BackendHTTPSettings) but not both"),
}

func validateServiceDefinition(eventRecorder record.EventRecorder, config *n.ApplicationGatewayPropertiesFormat, envVariables environment.EnvVariables, ingressList []*v1beta1.Ingress, serviceList []*v1.Service) error {
	// TODO(draychev): reuse newBackendIds() to get backendIDs oncehttps://github.com/Azure/application-gateway-kubernetes-ingress/pull/262 is merged
	backendIDs := make(map[backendIdentifier]interface{})
	for _, ingress := range ingressList {
		if ingress.Spec.Backend != nil {
			backendIDs[generateBackendID(ingress, nil, nil, ingress.Spec.Backend)] = nil
		}
		for ruleIdx := range ingress.Spec.Rules {
			rule := &ingress.Spec.Rules[ruleIdx]
			if rule.HTTP == nil {
				continue
			}
			for pathIdx := range rule.HTTP.Paths {
				path := &rule.HTTP.Paths[pathIdx]
				backendIDs[generateBackendID(ingress, rule, path, &path.Backend)] = nil
			}
		}
	}

	serviceSet := newServiceSet(&serviceList)
	for be := range backendIDs {
		if _, exists := serviceSet[be.serviceKey()]; !exists {
			logLine := fmt.Sprintf("Ingress %s/%s references non existent Service %s. Please correct the Service section of your Kubernetes YAML", be.Ingress.Namespace, be.Ingress.Name, be.serviceKey())
			eventRecorder.Event(be.Ingress, v1.EventTypeWarning, "IngressServiceTargetMatch", logLine)
			// NOTE: we could and should return errors.New(logLine)
			// However this could be enabled at a later point in time once we know with certainty taht there are no valid
			// scenarios where one could have Ingress pointing to a missing Service targets.
		}
	}
	return nil
}

func validateURLPathMaps(eventRecorder record.EventRecorder, config *n.ApplicationGatewayPropertiesFormat, envVariables environment.EnvVariables, ingressList []*v1beta1.Ingress, serviceList []*v1.Service) error {
	if config.URLPathMaps == nil {
		return nil
	}

	for _, pathMap := range *config.URLPathMaps {
		if len(*pathMap.PathRules) == 0 {
			// There are no paths. This is a rule of type "Basic"
			validRedirect := pathMap.DefaultRedirectConfiguration != nil
			validBackend := pathMap.DefaultBackendAddressPool != nil && pathMap.DefaultBackendHTTPSettings != nil

			if !validRedirect && !validBackend {
				// TODO(draychev): Emit an event for the appropriate object
				return validationErrors[errKeyNoDefaults]
			}

			if validRedirect && validBackend || !validRedirect && !validBackend {
				// TODO(draychev): Emit an event for the appropriate object
				return validationErrors[errKeyEitherDefaults]
			}

		} else {
			// There are paths defined. This is a rule of type "Path-based"
			for _, rule := range *pathMap.PathRules {
				validRedirect := rule.RedirectConfiguration != nil
				validBackend := rule.BackendAddressPool != nil && rule.BackendHTTPSettings != nil

				if !validRedirect && !validBackend {
					return validationErrors[errKeyNoBorR]
				}

				if validRedirect && validBackend || !validRedirect && !validBackend {
					return validationErrors[errKeyEitherBorR]
				}
			}

		}
	}
	return nil
}

// FatalValidateOnExistingConfig validates the existing configuration is valid for the specified setting of the controller.
func FatalValidateOnExistingConfig(eventRecorder record.EventRecorder, config *n.ApplicationGatewayPropertiesFormat, envVariables environment.EnvVariables) error {

	validators := []func(eventRecorder record.EventRecorder, config *network.ApplicationGatewayPropertiesFormat, envVariables environment.EnvVariables) error{}

	for _, fn := range validators {
		if err := fn(eventRecorder, config, envVariables); err != nil {
			return err
		}
	}

	return nil
}
