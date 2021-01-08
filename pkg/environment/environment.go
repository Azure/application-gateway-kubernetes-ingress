// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package environment

import (
	"os"
	"regexp"
	"strconv"

	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
)

const (
	// CloudProviderConfigLocationVarName is an environment variable name. This file is available on azure cluster.
	CloudProviderConfigLocationVarName = "AZURE_CLOUD_PROVIDER_LOCATION"

	// ClientIDVarName is an environment variable which stores the client id provided through user assigned identity
	ClientIDVarName = "AZURE_CLIENT_ID"

	// SubscriptionIDVarName is the name of the APPGW_SUBSCRIPTION_ID
	SubscriptionIDVarName = "APPGW_SUBSCRIPTION_ID"

	// ResourceGroupNameVarName is the name of the APPGW_RESOURCE_GROUP
	ResourceGroupNameVarName = "APPGW_RESOURCE_GROUP"

	// AppGwNameVarName is the name of the APPGW_NAME
	AppGwNameVarName = "APPGW_NAME"

	// AppGwSubnetNameVarName is the name of the APPGW_SUBNET_NAME
	AppGwSubnetNameVarName = "APPGW_SUBNET_NAME"

	// AppGwSubnetPrefixVarName is the name of the APPGW_SUBNET_PREFIX
	AppGwSubnetPrefixVarName = "APPGW_SUBNET_PREFIX"

	// AppGwResourceIDVarName is the name of the APPGW_RESOURCE_ID
	AppGwResourceIDVarName = "APPGW_RESOURCE_ID"

	// AppGwSubnetIDVarName is the name of the APPGW_SUBNET_ID
	AppGwSubnetIDVarName = "APPGW_SUBNET_ID"

	// AppGwSkuVarName is the sku of the AGW
	AppGwSkuVarName = "APPGW_SKU_NAME"

	// AuthLocationVarName is the name of the AZURE_AUTH_LOCATION
	AuthLocationVarName = "AZURE_AUTH_LOCATION"

	// WatchNamespaceVarName is the name of the KUBERNETES_WATCHNAMESPACE
	WatchNamespaceVarName = "KUBERNETES_WATCHNAMESPACE"

	// UsePrivateIPVarName is the name of the USE_PRIVATE_IP
	UsePrivateIPVarName = "USE_PRIVATE_IP"

	// VerbosityLevelVarName sets the level of klog verbosity should the CLI argument be blank
	VerbosityLevelVarName = "APPGW_VERBOSITY_LEVEL"

	// EnableBrownfieldDeploymentVarName is a feature flag enabling observation of {Managed,Prohibited}Target CRDs
	EnableBrownfieldDeploymentVarName = "APPGW_ENABLE_SHARED_APPGW"

	// UseAllowedTargetsBrownfieldDeploymentVarName is a feature flag enabling observation of AllowedTargets CRDs instead ProhibitedTarget CRDs
	UseAllowedTargetsBrownfieldDeploymentVarName = "APPGW_USE_ALLOWED_TARGETS"

	// EnableIstioIntegrationVarName is a feature flag enabling observation of Istio specific CRDs
	EnableIstioIntegrationVarName = "APPGW_ENABLE_ISTIO_INTEGRATION"

	// EnableSaveConfigToFileVarName is a feature flag, which enables saving the App Gwy config to disk.
	EnableSaveConfigToFileVarName = "APPGW_ENABLE_SAVE_CONFIG_TO_FILE"

	// EnablePanicOnPutErrorVarName is a feature flag.
	EnablePanicOnPutErrorVarName = "APPGW_ENABLE_PANIC_ON_PUT_ERROR"

	// EnableDeployAppGatewayVarName is a feature flag.
	EnableDeployAppGatewayVarName = "APPGW_ENABLE_DEPLOY"

	// HTTPServicePortVarName is an environment variable name.
	HTTPServicePortVarName = "HTTP_SERVICE_PORT"

	// AGICPodNameVarName is an environment variable name.
	AGICPodNameVarName = "AGIC_POD_NAME"

	// AGICPodNamespaceVarName is an environment variable name.
	AGICPodNamespaceVarName = "AGIC_POD_NAMESPACE"

	// UseManagedIdentityForPodVarName is an environment variable name.
	UseManagedIdentityForPodVarName = "USE_MANAGED_IDENTITY_FOR_POD"

	// AttachWAFPolicyToListenerVarName is an environment variable name.
	AttachWAFPolicyToListenerVarName = "ATTACH_WAF_POLICY_TO_LISTENER"

	// HostedOnUnderlayVarName  is an environment variable name.
	HostedOnUnderlayVarName = "HOSTED_ON_UNDERLAY"

	// ReconcilePeriodSecondsVarName is an environment variable to control reconcile period for the AGIC.
	ReconcilePeriodSecondsVarName = "RECONCILE_PERIOD_SECONDS"

	// IngressClass is an environment variable
	IngressClass = "INGRESS_CLASS"
)

var (
	portNumberValidator = regexp.MustCompile(`^[0-9]{4,5}$`)
	skuValidator        = regexp.MustCompile(`WAF_v2|Standard_v2`)
	boolValidator       = regexp.MustCompile(`^(?i)(true|false)$`)
)

// EnvVariables is a struct storing values for environment variables.
type EnvVariables struct {
	CloudProviderConfigLocation           string
	ClientID                              string
	SubscriptionID                        string
	ResourceGroupName                     string
	AppGwName                             string
	AppGwSubnetName                       string
	AppGwSubnetPrefix                     string
	AppGwResourceID                       string
	AppGwSubnetID                         string
	AppGwSkuName                          string
	AuthLocation                          string
	IngressClass                          string
	WatchNamespace                        string
	UsePrivateIP                          string
	VerbosityLevel                        string
	AGICPodName                           string
	AGICPodNamespace                      string
	EnableBrownfieldDeployment            bool
	UseAllowedTargetsBrownfieldDeployment bool
	EnableIstioIntegration                bool
	EnableSaveConfigToFile                bool
	EnablePanicOnPutError                 bool
	EnableDeployAppGateway                bool
	UseManagedIdentityForPod              bool
	HTTPServicePort                       string
	AttachWAFPolicyToListener             bool
	HostedOnUnderlay                      bool
	ReconcilePeriodSeconds                string
}

// Consolidate sets defaults and missing values using cpConfig
func (env *EnvVariables) Consolidate(cpConfig *azure.CloudProviderConfig) {
	// adjust env variable
	if env.AppGwResourceID != "" {
		subscriptionID, resourceGroupName, applicationGatewayName := azure.ParseResourceID(env.AppGwResourceID)
		env.SubscriptionID = string(subscriptionID)
		env.ResourceGroupName = string(resourceGroupName)
		env.AppGwName = string(applicationGatewayName)
	}

	// Set using cloud provider config
	if cpConfig != nil {
		if env.SubscriptionID == "" {
			env.SubscriptionID = string(cpConfig.SubscriptionID)
		}

		if env.ResourceGroupName == "" {
			env.ResourceGroupName = string(cpConfig.ResourceGroup)
		}
	}

	// Set defaults
	if env.AppGwSubnetName == "" {
		env.AppGwSubnetName = env.AppGwName + "-subnet"
	}
}

// GetEnv returns values for defined environment variables for Ingress Controller.
func GetEnv() EnvVariables {
	env := EnvVariables{
		CloudProviderConfigLocation:           os.Getenv(CloudProviderConfigLocationVarName),
		ClientID:                              os.Getenv(ClientIDVarName),
		SubscriptionID:                        os.Getenv(SubscriptionIDVarName),
		ResourceGroupName:                     os.Getenv(ResourceGroupNameVarName),
		AppGwName:                             os.Getenv(AppGwNameVarName),
		AppGwSubnetName:                       os.Getenv(AppGwSubnetNameVarName),
		AppGwSubnetPrefix:                     os.Getenv(AppGwSubnetPrefixVarName),
		AppGwResourceID:                       os.Getenv(AppGwResourceIDVarName),
		AppGwSubnetID:                         os.Getenv(AppGwSubnetIDVarName),
		AppGwSkuName:                          GetEnvironmentVariable(AppGwSkuVarName, "Standard_v2", skuValidator),
		AuthLocation:                          os.Getenv(AuthLocationVarName),
		IngressClass:                          os.Getenv(IngressClass),
		WatchNamespace:                        os.Getenv(WatchNamespaceVarName),
		UsePrivateIP:                          os.Getenv(UsePrivateIPVarName),
		VerbosityLevel:                        os.Getenv(VerbosityLevelVarName),
		AGICPodName:                           os.Getenv(AGICPodNameVarName),
		AGICPodNamespace:                      os.Getenv(AGICPodNamespaceVarName),
		EnableBrownfieldDeployment:            GetEnvironmentVariable(EnableBrownfieldDeploymentVarName, "false", boolValidator) == "true",
		UseAllowedTargetsBrownfieldDeployment: GetEnvironmentVariable(UseAllowedTargetsBrownfieldDeploymentVarName, "false", boolValidator) == "true",
		EnableIstioIntegration:                GetEnvironmentVariable(EnableIstioIntegrationVarName, "false", boolValidator) == "true",
		EnableSaveConfigToFile:                GetEnvironmentVariable(EnableSaveConfigToFileVarName, "false", boolValidator) == "true",
		EnablePanicOnPutError:                 GetEnvironmentVariable(EnablePanicOnPutErrorVarName, "false", boolValidator) == "true",
		EnableDeployAppGateway:                GetEnvironmentVariable(EnableDeployAppGatewayVarName, "false", boolValidator) == "true",
		UseManagedIdentityForPod:              GetEnvironmentVariable(UseManagedIdentityForPodVarName, "false", boolValidator) == "true",
		HTTPServicePort:                       GetEnvironmentVariable(HTTPServicePortVarName, "8123", portNumberValidator),
		AttachWAFPolicyToListener:             GetEnvironmentVariable(AttachWAFPolicyToListenerVarName, "false", boolValidator) == "true",
		HostedOnUnderlay:                      GetEnvironmentVariable(HostedOnUnderlayVarName, "false", boolValidator) == "true",
		ReconcilePeriodSeconds:                os.Getenv(ReconcilePeriodSecondsVarName),
	}

	return env
}

// ValidateEnv validates environment variables.
func ValidateEnv(env EnvVariables) error {
	if env.EnableDeployAppGateway {
		// we should not allow applicationGatewayID in create case
		if len(env.AppGwResourceID) != 0 {
			return controllererrors.NewError(
				controllererrors.ErrorNotAllowedApplicationGatewayID,
				"Please provide provide APPGW_NAME (helm var name: .appgw.name) instead of APPGW_RESOURCE_ID (helm var name: .appgw.applicationGatewayID). "+
					"You can also provided APPGW_SUBSCRIPTION_ID and APPGW_RESOURCE_GROUP",
			)
		}

		// if deploy is true, we need applicationGatewayName
		if len(env.AppGwName) == 0 {
			return controllererrors.NewError(
				controllererrors.ErrorMissingApplicationGatewayName,
				"Missing required Environment variables: AGIC requires APPGW_NAME (helm var name: appgw.name) to deploy Application Gateway",
			)
		}

		// we need one of subnetID and subnetPrefix. We generate a subnetName if it is not provided.
		if len(env.AppGwSubnetID) == 0 && len(env.AppGwSubnetPrefix) == 0 {
			// when create is true, then either we should have env.AppGwSubnetID or env.AppGwSubnetPrefix
			return controllererrors.NewError(
				controllererrors.ErrorMissingSubnetInfo,
				"Missing required Environment variables: "+
					"AGIC requires APPGW_SUBNET_PREFIX (helm var name: appgw.subnetPrefix) or APPGW_SUBNET_ID (helm var name: appgw.subnetID) of an existing subnet. "+
					"If subnetPrefix is specified, AGIC will look up a subnet with matching address prefix in the AKS cluster vnet. "+
					"If a subnet is not found, then a new subnet will be created. This will be used to deploy the Application Gateway",
			)

		}
	} else {
		// if deploy is false, we need one of appgw name or resource id
		if len(env.AppGwName) == 0 && len(env.AppGwResourceID) == 0 {
			return controllererrors.NewError(
				controllererrors.ErrorMissingApplicationGatewayNameOrApplicationGatewayID,
				"Missing required Environment variables: "+
					"Provide atleast provide APPGW_NAME (helm var name: .appgw.name) or APPGW_RESOURCE_ID (helm var name: .appgw.applicationGatewayID). "+
					"If providing APPGW_NAME, You can also provided APPGW_SUBSCRIPTION_ID (helm var name: .appgw.subscriptionId) and APPGW_RESOURCE_GROUP (helm var name: .appgw.resourceGroup)",
			)
		}
	}

	if env.WatchNamespace == "" {
		klog.V(1).Infof("%s is not set. Watching all available namespaces.", WatchNamespaceVarName)
	}

	if env.ReconcilePeriodSeconds != "" {
		reconcilePeriodSeconds, err := strconv.Atoi(env.ReconcilePeriodSeconds)
		if err != nil {
			return controllererrors.NewErrorWithInnerError(
				controllererrors.ErrorInvalidReconcilePeriod,
				err,
				"Please make sure that RECONCILE_PERIOD_SECONDS (helm var name: .reconcilePeriodSeconds) is an integer. Range: (30 - 300)",
			)
		}

		if reconcilePeriodSeconds < 30 || reconcilePeriodSeconds > 300 {
			return controllererrors.NewError(
				controllererrors.ErrorInvalidReconcilePeriod,
				"Please make sure that RECONCILE_PERIOD_SECONDS (helm var name: .reconcilePeriodSeconds) is an integer. Range: (30 - 300)",
			)
		}
	}

	return nil
}

// GetEnvironmentVariable is an augmentation of os.Getenv, providing it with a default value.
func GetEnvironmentVariable(environmentVariable, defaultValue string, validator *regexp.Regexp) string {
	if value, ok := os.LookupEnv(environmentVariable); ok {
		if validator == nil {
			return value
		}
		if validator.MatchString(value) {
			return value
		}
		klog.Errorf("Environment variable %s contains a value which does not pass validation filter; Using default value: %s", environmentVariable, defaultValue)
	}
	return defaultValue
}
