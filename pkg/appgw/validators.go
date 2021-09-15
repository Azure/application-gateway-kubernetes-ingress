// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package appgw

import (
	"fmt"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	v1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
)

func validateServiceDefinition(eventRecorder record.EventRecorder, config *n.ApplicationGatewayPropertiesFormat, envVariables environment.EnvVariables, ingressList []*networking.Ingress, serviceList []*v1.Service) error {
	// TODO(draychev): reuse newBackendIds() to get backendIDs oncehttps://github.com/Azure/application-gateway-kubernetes-ingress/pull/262 is merged
	backendIDs := make(map[backendIdentifier]interface{})
	for _, ingress := range ingressList {
		if ingress.Spec.DefaultBackend != nil {
			backendIDs[generateBackendID(ingress, nil, nil, ingress.Spec.DefaultBackend, ingress.Spec.DefaultBackend.Service.Name)] = nil
		}
		for ruleIdx := range ingress.Spec.Rules {
			rule := &ingress.Spec.Rules[ruleIdx]
			if rule.HTTP == nil {
				continue
			}
			for pathIdx := range rule.HTTP.Paths {
				if ingress.Spec.DefaultBackend == nil {
					continue
				}
				path := &rule.HTTP.Paths[pathIdx]
				backendIDs[generateBackendID(ingress, rule, path, &path.Backend, path.Backend.Service.Name)] = nil
			}
		}
	}

	serviceSet := newServiceSet(&serviceList)
	for be := range backendIDs {
		if _, exists := serviceSet[be.serviceKey()]; !exists {
			logLine := fmt.Sprintf("Ingress %s/%s references non existent Service %s. Please correct the Service section of your Kubernetes YAML", be.Ingress.Namespace, be.Ingress.Name, be.serviceKey())
			eventRecorder.Event(be.Ingress, v1.EventTypeWarning, events.ReasonIngressServiceTargetMatch, logLine)
			// NOTE: We could and should return a new error here.
			// However this could be enabled at a later point in time once we know with certainty that there are no valid
			// scenarios where one could have Ingress pointing to a missing Service targets.
		}
	}
	return nil
}

func validateURLPathMaps(eventRecorder record.EventRecorder, config *n.ApplicationGatewayPropertiesFormat, envVariables environment.EnvVariables, ingressList []*networking.Ingress, serviceList []*v1.Service) error {
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
				e := controllererrors.NewErrorf(
					controllererrors.ErrorNoDefaults,
					"URL path map '%s' needs either a default backend pool or default redirect", *pathMap.Name,
				)
				return e
			}

			if validRedirect && validBackend || !validRedirect && !validBackend {
				// TODO(draychev): Emit an event for the appropriate object
				e := controllererrors.NewErrorf(
					controllererrors.ErrorEitherDefaults,
					"URL path map '%s' needs only one of default backend pool or default redirect", *pathMap.Name,
				)
				return e
			}

		} else {
			// There are paths defined. This is a rule of type "Path-based"
			for _, rule := range *pathMap.PathRules {
				validRedirect := rule.RedirectConfiguration != nil
				validBackend := rule.BackendAddressPool != nil && rule.BackendHTTPSettings != nil

				if !validRedirect && !validBackend {
					e := controllererrors.NewErrorf(
						controllererrors.ErrorNoBackendorRedirect,
						"Path Rule '%s' needs either a default backend pool or default redirect", *rule.Name,
					)
					return e
				}

				if validRedirect && validBackend || !validRedirect && !validBackend {
					e := controllererrors.NewErrorf(
						controllererrors.ErrorEitherBackendorRedirect,
						"Path Rule '%s' needs either a default backend pool or default redirect", *rule.Name,
					)
					return e
				}
			}

		}
	}
	return nil
}

func validateFrontendIPConfiguration(eventRecorder record.EventRecorder, config n.ApplicationGatewayPropertiesFormat, envVariables environment.EnvVariables) error {
	privateIPPresent := false
	publicIPPresent := false
	var jsonConfigs []string
	for _, ip := range *config.FrontendIPConfigurations {
		if jsonConf, err := ip.MarshalJSON(); err != nil {
			klog.Error("Could not marshall IP configuration:", *ip.ID, err)
		} else {
			jsonConfigs = append(jsonConfigs, string(jsonConf))
		}

		privateIPPresent = privateIPPresent ||
			(ip.ApplicationGatewayFrontendIPConfigurationPropertiesFormat != nil && ip.PrivateIPAddress != nil)
		publicIPPresent = publicIPPresent ||
			(ip.ApplicationGatewayFrontendIPConfigurationPropertiesFormat != nil && ip.PublicIPAddress != nil)
	}

	if envVariables.UsePrivateIP && !privateIPPresent {
		e := controllererrors.NewError(
			controllererrors.ErrorNoPrivateIP,
			"Applcation Gateway doesn't have a private IP. Either add a private IP or set usePrivateIP flag to false in helm config.",
		)
		return e
	}

	if !publicIPPresent {
		e := controllererrors.NewError(
			controllererrors.ErrorNoPublicIP,
			"Applcation Gateway doesn't have a public IP",
		)
		return e
	}

	return nil
}

// FatalValidateOnExistingConfig validates the existing configuration is valid for the specified setting of the controller.
func FatalValidateOnExistingConfig(eventRecorder record.EventRecorder, config *n.ApplicationGatewayPropertiesFormat, envVariables environment.EnvVariables) error {
	if config == nil {
		e := controllererrors.NewError(
			controllererrors.ErrorEmptyConfig,
			"Application Gateway configuration should not be empty.",
		)
		return e
	}

	validators := []func(eventRecorder record.EventRecorder, config n.ApplicationGatewayPropertiesFormat, envVariables environment.EnvVariables) error{
		validateFrontendIPConfiguration,
	}

	for _, fn := range validators {
		if err := fn(eventRecorder, *config, envVariables); err != nil {
			return err
		}
	}

	return nil
}
