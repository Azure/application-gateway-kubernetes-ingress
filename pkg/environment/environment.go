// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package environment

import (
	"errors"
	"os"
	"regexp"

	"github.com/golang/glog"
)

const (
	// AzContextLocationVarName is an environment variable name. This file is available on azure cluster.
	AzContextLocationVarName = "AZURE_CONTEXT_LOCATION"

	// SubscriptionIDVarName is the name of the APPGW_SUBSCRIPTION_ID
	SubscriptionIDVarName = "APPGW_SUBSCRIPTION_ID"

	// ResourceGroupNameVarName is the name of the APPGW_RESOURCE_GROUP
	ResourceGroupNameVarName = "APPGW_RESOURCE_GROUP"

	// AppGwNameVarName is the name of the APPGW_NAME
	AppGwNameVarName = "APPGW_NAME"

	// AppGwSubnetPrefixVarName is the name of the APPGW_SUBNETPREFIX
	AppGwSubnetPrefixVarName = "APPGW_SUBNETPREFIX"

	// AppGwSubnetIDVarName is the name of the APPGW_SUBNETID
	AppGwSubnetIDVarName = "APPGW_SUBNETID"

	// AppGwSubnetNameVarName is the name of the APPGW_SUBNETNAME
	AppGwSubnetNameVarName = "APPGW_SUBNETNAME"

	// ReleaseNameVarName is the name of the RELEASE_NAME
	ReleaseNameVarName = "RELEASE_NAME"

	// AuthLocationVarName is the name of the AZURE_AUTH_LOCATION
	AuthLocationVarName = "AZURE_AUTH_LOCATION"

	// WatchNamespaceVarName is the name of the KUBERNETES_WATCHNAMESPACE
	WatchNamespaceVarName = "KUBERNETES_WATCHNAMESPACE"

	// UsePrivateIPVarName is the name of the USE_PRIVATE_IP
	UsePrivateIPVarName = "USE_PRIVATE_IP"

	// VerbosityLevelVarName sets the level of glog verbosity should the CLI argument be blank
	VerbosityLevelVarName = "APPGW_VERBOSITY_LEVEL"

	// EnableBrownfieldDeploymentVarName is a feature flag enabling observation of {Managed,Prohibited}Target CRDs
	EnableBrownfieldDeploymentVarName = "APPGW_ENABLE_SHARED_APPGW"

	// EnableIstioIntegrationVarName is a feature flag enabling observation of Istio specific CRDs
	EnableIstioIntegrationVarName = "APPGW_ENABLE_ISTIO_INTEGRATION"

	// EnableSaveConfigToFileVarName is a feature flag, which enables saving the App Gwy config to disk.
	EnableSaveConfigToFileVarName = "APPGW_ENABLE_SAVE_CONFIG_TO_FILE"

	// EnablePanicOnPutErrorVarName is a feature flag.
	EnablePanicOnPutErrorVarName = "APPGW_ENABLE_PANIC_ON_PUT_ERROR"

	// EnableDeployAppGatewayVarName is a feature flag.
	EnableDeployAppGatewayVarName = "APPGW_ENABLE_DEPLOY_APPGATEWAY"

	// HTTPServicePortVarName is an environment variable name.
	HTTPServicePortVarName = "HTTP_SERVICE_PORT"

	// AGICPodNameVarName is an environment variable name.
	AGICPodNameVarName = "AGIC_POD_NAME"

	// AGICPodNamespaceVarName is an environment variable name.
	AGICPodNamespaceVarName = "AGIC_POD_NAMESPACE"

	// UseManagedIdentityForPodVarName is an environment variable name.
	UseManagedIdentityForPodVarName = "USE_MANAGED_IDENTITY_FOR_POD"
)

// EnvVariables is a struct storing values for environment variables.
type EnvVariables struct {
	AzContextLocation          string
	SubscriptionID             string
	ResourceGroupName          string
	AppGwName                  string
	AppGwSubnetPrefix          string
	AppGwSubnetID              string
	AppGwSubnetName            string
	ReleaseName                string
	AuthLocation               string
	WatchNamespace             string
	UsePrivateIP               string
	VerbosityLevel             string
	AGICPodName                string
	AGICPodNamespace           string
	EnableBrownfieldDeployment bool
	EnableIstioIntegration     bool
	EnableSaveConfigToFile     bool
	EnablePanicOnPutError      bool
	EnableDeployAppGateway     bool
	UseManagedIdentityForPod   bool
	HTTPServicePort            string
}

var portNumberValidator = regexp.MustCompile(`^[0-9]{4,5}$`)
var boolValidator = regexp.MustCompile(`^(?i)(true|false)$`)

// GetEnv returns values for defined environment variables for Ingress Controller.
func GetEnv() EnvVariables {
	env := EnvVariables{
		AzContextLocation:          os.Getenv(AzContextLocationVarName),
		SubscriptionID:             os.Getenv(SubscriptionIDVarName),
		ResourceGroupName:          os.Getenv(ResourceGroupNameVarName),
		AppGwName:                  os.Getenv(AppGwNameVarName),
		AppGwSubnetPrefix:          os.Getenv(AppGwSubnetPrefixVarName),
		AppGwSubnetID:              os.Getenv(AppGwSubnetIDVarName),
		AppGwSubnetName:            os.Getenv(AppGwSubnetNameVarName),
		ReleaseName:                os.Getenv(ReleaseNameVarName),
		AuthLocation:               os.Getenv(AuthLocationVarName),
		WatchNamespace:             os.Getenv(WatchNamespaceVarName),
		UsePrivateIP:               os.Getenv(UsePrivateIPVarName),
		VerbosityLevel:             os.Getenv(VerbosityLevelVarName),
		AGICPodName:                os.Getenv(AGICPodNameVarName),
		AGICPodNamespace:           os.Getenv(AGICPodNamespaceVarName),
		EnableBrownfieldDeployment: GetEnvironmentVariable(EnableBrownfieldDeploymentVarName, "false", boolValidator) == "true",
		EnableIstioIntegration:     GetEnvironmentVariable(EnableIstioIntegrationVarName, "false", boolValidator) == "true",
		EnableSaveConfigToFile:     GetEnvironmentVariable(EnableSaveConfigToFileVarName, "false", boolValidator) == "true",
		EnablePanicOnPutError:      GetEnvironmentVariable(EnablePanicOnPutErrorVarName, "false", boolValidator) == "true",
		EnableDeployAppGateway:     GetEnvironmentVariable(EnableDeployAppGatewayVarName, "false", boolValidator) == "true",
		UseManagedIdentityForPod:   GetEnvironmentVariable(UseManagedIdentityForPodVarName, "false", boolValidator) == "true",
		HTTPServicePort:            GetEnvironmentVariable(HTTPServicePortVarName, "8123", portNumberValidator),
	}

	return env
}

// ValidateEnv validates environment variables.
func ValidateEnv(env EnvVariables) error {
	if len(env.AppGwName) == 0 {
		return errors.New("Missing required Environment variables: Provide atleast provide APPGW_NAME. You can also provided APPGW_SUBSCRIPTION_ID and APPGW_RESOURCE_GROUP (ENVT001)")
	}
	if env.EnableDeployAppGateway && len(env.AppGwSubnetID) == 0 && len(env.AppGwSubnetPrefix) == 0 && len(env.AppGwSubnetName) == 0 {
		// when create is true, then either we should have env.AppGwSubnetID or env.AppGwSubnetPrefix
		return errors.New("Missing required Environment variables: Please provide APPGW_SUBNETNAME or APPGW_SUBNETID of an existing subnet. If you want AGIC to optionally create a new subnet, then also provide APPGW_SUBNETPREFIX (ENVT002)")
	}

	if env.WatchNamespace == "" {
		glog.V(1).Infof("%s is not set. Watching all available namespaces.", WatchNamespaceVarName)
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
		glog.Errorf("Environment variable %s contains a value which does not pass validation filter; Using default value: %s", environmentVariable, defaultValue)
	}
	return defaultValue
}
