// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import (
	"context"
	"flag"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/annotations"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controller"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned"
	istio "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/version"
)

const (
	verbosityFlag     = "verbosity"
	maxAuthRetryCount = 10
	tenSeconds        = 10 * time.Second
	thirtySeconds     = 30 * time.Second
)

var (
	flags = pflag.NewFlagSet(`appgw-ingress`, pflag.ExitOnError)

	inCluster = flags.Bool("in-cluster", true,
		"If running in a Kubernetes cluster, use the pod secrets for creating a Kubernetes client. Optional.")

	apiServerHost = flags.String("apiserver-host", "",
		"The address of the Kubernetes apiserver. Optional if running in cluster; if omitted, local discovery is attempted.")

	kubeConfigFile = flags.String("kubeconfig", "",
		"Path to kubeconfig file with authorization and master location information.")

	resyncPeriod = flags.Duration("sync-period", thirtySeconds,
		"Interval at which to re-list and confirm cloud resources.")

	versionInfo = flags.Bool("version", false, "Print version")

	verbosity = flags.Int(verbosityFlag, 1, "Set logging verbosity level")
)

func main() {
	// Log output is buffered... Calling Flush before exiting guarantees all log output is written.
	defer glog.Flush()
	if err := flags.Parse(os.Args); err != nil {
		glog.Fatal("Error parsing command line arguments:", err)
	}

	env := environment.GetEnv()
	environment.ValidateEnv(env)

	verbosity = to.IntPtr(getVerbosity(*verbosity, env.VerbosityLevel))

	if *versionInfo {
		version.PrintVersionAndExit()
	}

	// Workaround for "ERROR: logging before flag.Parse"
	// See: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	_ = flag.CommandLine.Parse([]string{})
	_ = flag.Lookup("logtostderr").Value.Set("true")
	_ = flag.Set("v", strconv.Itoa(*verbosity))

	// initialize clients and dependencies
	appGwClient, err := initAppGwClient(env)
	if err != nil {
		glog.Fatal("Error creating Azure client: ", err)
	}

	appGwIdentifier := appgw.Identifier{
		SubscriptionID: env.SubscriptionID,
		ResourceGroup:  env.ResourceGroupName,
		AppGwName:      env.AppGwName,
	}

	apiConfig := getKubeClientConfig()
	kubeClient := kubernetes.NewForConfigOrDie(apiConfig)
	crdClient := versioned.NewForConfigOrDie(apiConfig)
	istioCrdClient := istio.NewForConfigOrDie(apiConfig)
	recorder := getEventRecorder(kubeClient)
	namespaces := getNamespacesToWatch(env.WatchNamespace)
	k8sContext := k8scontext.NewContext(kubeClient, crdClient, istioCrdClient, namespaces, *resyncPeriod)

	// namespace validations
	validateNamespaces(namespaces, kubeClient) // side-effect: will panic on non-existent namespace
	if len(namespaces) == 0 {
		glog.Info("Ingress Controller will observe all namespaces.")
	} else {
		glog.Info("Ingress Controller will observe the following namespaces:", strings.Join(namespaces, ","))
	}

	// fatal config validations
	appGw, _ := appGwClient.Get(context.Background(), env.ResourceGroupName, env.AppGwName)
	if err := appgw.FatalValidateOnExistingConfig(recorder, appGw.ApplicationGatewayPropertiesFormat, env); err != nil {
		glog.Fatal("Got a fatal validation error on existing Application Gateway config. Please update Application Gateway or the controller's helm config. Error:", err)
	}

	// initiliaze controller
	appGwIngressController := controller.NewAppGwIngressController(*appGwClient, appGwIdentifier, k8sContext, recorder)

	// start controller
	appGwIngressController.Start(env)
}

func validateNamespaces(namespaces []string, kubeClient *kubernetes.Clientset) {
	var nonExistent []string
	for _, ns := range namespaces {
		if _, err := kubeClient.CoreV1().Namespaces().Get(ns, metav1.GetOptions{}); err != nil {
			nonExistent = append(nonExistent, ns)
		}
	}
	if len(nonExistent) > 0 {
		glog.Fatalf("Error creating informers; Namespaces do not exist or Ingress Controller has no access to: %v", strings.Join(nonExistent, ","))
	}
}

func getNamespacesToWatch(namespaceEnvVar string) []string {
	// Returning an empty array effectively switches Ingress Controller
	// in a mode of observing all accessible namespaces.
	if namespaceEnvVar == "" {
		return []string{}
	}

	// Namespaces (DNS-1123 label) can have lower case alphanumeric characters or '-'
	// Commas are safe to use as a separator
	if strings.Contains(namespaceEnvVar, ",") {
		var namespaces []string
		for _, ns := range strings.Split(namespaceEnvVar, ",") {
			if len(ns) > 0 {
				namespaces = append(namespaces, strings.TrimSpace(ns))
			}
		}
		sort.Strings(namespaces)
		return namespaces
	}
	return []string{namespaceEnvVar}
}

func initAppGwClient(env environment.EnvVariables) (*n.ApplicationGatewaysClient, error) {
	appGwClient := n.NewApplicationGatewaysClient(env.SubscriptionID)
	if err := waitForAzureAuth(env, &appGwClient); err != nil {
		return nil, err
	}

	return &appGwClient, nil
}

func waitForAzureAuth(env environment.EnvVariables, client *n.ApplicationGatewaysClient) error {
	var response n.ApplicationGateway
	var err error
	for counter := 0; counter <= maxAuthRetryCount; counter++ {
		// Fetch a new token
		if client.Authorizer, err = getAzAuth(env); err != nil || client.Authorizer == nil {
			glog.Fatal("Error creating Azure client", err)
		}

		// Get Application Gateway
		response, err = client.Get(context.Background(), env.ResourceGroupName, env.AppGwName)
		if err == nil {
			return nil
		}

		// Tries remaining
		if counter < maxAuthRetryCount {
			glog.Errorf("Failed fetching config for App Gateway instance %s. Will retry in %v. ARM Error: %s", env.AppGwName, tenSeconds, err)
			time.Sleep(tenSeconds)
		}
	}

	// Reasons for 403 errors
	if response.Response.StatusCode == 403 {
		infoLine := "Possible reasons:"
		infoLine += " AKS Service Principal requires 'Managed Identity Operator' access on Controller Identity;"
		infoLine += " 'identityResourceID' and/or 'identityClientID' are incorrect in the Helm config;"
		infoLine += " AGIC Identity requires 'Contributor' access on Application Gateway and 'Reader' access on Application Gateway's Resource Group;"
		glog.Error(infoLine)
	}

	if response.Response.StatusCode != 200 {
		// for example, getting 401. This is not expected as we are getting a token before making the call.
		glog.Error("Recieved an unexpected status code from ARM while getting App Gateway: ", response.Response.StatusCode)
		return ErrUnexpectedARMStatusCode
	}

	return err
}

func getAzAuth(vars environment.EnvVariables) (autorest.Authorizer, error) {
	if vars.AuthLocation == "" {
		// requires aad-pod-identity to be deployed in the AKS cluster
		// see https://github.com/Azure/aad-pod-identity for more information
		glog.V(1).Info("Creating authorizer from Azure Managed Service Identity")
		return auth.NewAuthorizerFromEnvironment()
	}
	glog.V(1).Infof("Creating authorizer from file referenced by %s environment variable: %s", environment.AuthLocationVarName, vars.AuthLocation)
	return auth.NewAuthorizerFromFile(n.DefaultBaseURI)
}

func getKubeClientConfig() *rest.Config {
	if *inCluster {
		config, err := rest.InClusterConfig()
		if err != nil {
			glog.Fatal("Error creating client configuration:", err)
		}
		return config
	}

	if *apiServerHost == "" {
		glog.Fatal("when not running in a cluster you must specify --apiserver-host")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfigFile)

	if err != nil {
		glog.Fatal("error creating client configuration:", err)
	}

	return config
}

func getEventRecorder(kubeClient kubernetes.Interface) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartLogging(glog.V(3).Infof)
	sink := &typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")}
	eventBroadcaster.StartRecordingToSink(sink)
	hostname, err := os.Hostname()
	if err != nil {
		glog.Error("Could not obtain host name from the operating system", err)
		hostname = "unknown-hostname"
	}
	source := v1.EventSource{
		Component: annotations.ApplicationGatewayIngressClass,
		Host:      hostname,
	}
	return eventBroadcaster.NewRecorder(scheme.Scheme, source)
}

func getVerbosity(flagVerbosity int, envVerbosity string) int {
	envVerbosityInt, err := strconv.Atoi(envVerbosity)
	if err != nil {
		glog.Infof("Using verbosity level %d from CLI flag %s", flagVerbosity, verbosityFlag)
		return flagVerbosity
	}
	glog.Infof("Using verbosity level %d from environment variable %s", envVerbosityInt, environment.VerbosityLevelVarName)
	return envVerbosityInt
}
