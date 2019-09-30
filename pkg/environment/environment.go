// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package environment

import (
	"os"
	"regexp"

	"github.com/golang/glog"
)

const (
	// SubscriptionIDVarName is the name of the APPGW_SUBSCRIPTION_ID
	SubscriptionIDVarName = "APPGW_SUBSCRIPTION_ID"

	// ResourceGroupNameVarName is the name of the APPGW_RESOURCE_GROUP
	ResourceGroupNameVarName = "APPGW_RESOURCE_GROUP"

	// AppGwNameVarName is the name of the APPGW_NAME
	AppGwNameVarName = "APPGW_NAME"

	// AppGwSubnetIDVarName is the name of the APPGW_SUBNET_ID
	AppGwSubnetIDVarName = "APPGW_SUBNETID"

	// AppGwSubnetIDVarName is the name of the APPGW_SUBNET_ID
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

	// HTTPServicePortVarName is an environment variable name.
	HTTPServicePortVarName = "HTTP_SERVICE_PORT"

	// AGICPodNameVarName is an environment variable name.
	AGICPodNameVarName = "AGIC_POD_NAME"

	// AGICPodNamespaceVarName is an environment variable name.
	AGICPodNamespaceVarName = "AGIC_POD_NAMESPACE"
)

// EnvVariables is a struct storing values for environment variables.
type EnvVariables struct {
	SubscriptionID             string
	ResourceGroupName          string
	AppGwName                  string
	AppGwSubnetID              string
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
	HTTPServicePort            string
}

var portNumberValidator = regexp.MustCompile(`^[0-9]{4,5}$`)
var boolValidator = regexp.MustCompile(`^(?i)(true|false)$`)

// GetEnv returns values for defined environment variables for Ingress Controller.
func GetEnv() EnvVariables {
	env := EnvVariables{
		SubscriptionID:             os.Getenv(SubscriptionIDVarName),
		ResourceGroupName:          os.Getenv(ResourceGroupNameVarName),
		AppGwName:                  os.Getenv(AppGwNameVarName),
		AppGwSubnetID:              os.Getenv(AppGwSubnetIDVarName),
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
		HTTPServicePort:            GetEnvironmentVariable(HTTPServicePortVarName, "8123", portNumberValidator),
	}

	return env
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
