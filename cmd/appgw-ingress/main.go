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

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
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
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/version"
)

var (
	flags = pflag.NewFlagSet(`appgw-ingress`, pflag.ExitOnError)

	inCluster = flags.Bool("in-cluster", true,
		"If running in a Kubernetes cluster, use the pod secrets for creating a Kubernetes client. Optional.")

	apiServerHost = flags.String("apiserver-host", "",
		"The address of the Kubernetes apiserver. Optional if running in cluster; if omitted, local discovery is attempted.")

	kubeConfigFile = flags.String("kubeconfig", "",
		"Path to kubeconfig file with authorization and master location information.")

	resyncPeriod = flags.Duration("sync-period", 30*time.Second,
		"Interval at which to re-list and confirm cloud resources.")

	versionInfo = flags.Bool("version", false, "Print version")

	verbosity = flags.Int("verbosity", 1, "Set logging verbosity level")
)

func main() {
	// Log output is buffered... Calling Flush before exiting guarantees all log output is written.
	defer glog.Flush()
	if err := flags.Parse(os.Args); err != nil {
		glog.Fatal("Error parsing command line arguments:", err)
	}

	glog.Infof("Logging at verbosity level %d", *verbosity)

	if *versionInfo {
		version.PrintVersionAndExit()
	}

	// Workaround for "ERROR: logging before flag.Parse"
	// See: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	_ = flag.CommandLine.Parse([]string{})
	_ = flag.Lookup("logtostderr").Value.Set("true")
	_ = flag.Set("v", strconv.Itoa(*verbosity))

	env := environment.GetEnv()
	environment.ValidateEnv(env)

	appGwClient := network.NewApplicationGatewaysClient(env.SubscriptionID)

	var err error
	if appGwClient.Authorizer, err = getAzAuth(env); err != nil || appGwClient.Authorizer == nil {
		glog.Fatal("Error creating Azure client", err)
	}

	waitForAzureAuth(env, appGwClient)

	appGwIdentifier := appgw.Identifier{
		SubscriptionID: env.SubscriptionID,
		ResourceGroup:  env.ResourceGroupName,
		AppGwName:      env.AppGwName,
	}

	kubeClient, err := kubernetes.NewForConfig(getKubeClientConfig())
	if err != nil {
		glog.Fatal("Error creating Kubernetes client: ", err)
	}
	namespaces := getNamespacesToWatch(env.WatchNamespace)
	validateNamespaces(namespaces, kubeClient) // side-effect: will panic on non-existent namespace
	glog.Info("Ingress Controller will observe the following namespaces:", strings.Join(namespaces, ","))

	recorder := getEventRecorder(kubeClient)

	// Run fatal validations
	appGw, _ := appGwClient.Get(context.Background(), env.ResourceGroupName, env.AppGwName)
	if err := appgw.FatalValidateOnExistingConfig(recorder, appGw.ApplicationGatewayPropertiesFormat, env); err != nil {
		glog.Fatal("Got a fatal validation error on existing Application Gateway config. Please update Application Gateway or the controller's helm config.", err)
	}

	k8sContext := k8scontext.NewContext(kubeClient, namespaces, *resyncPeriod)

	go controller.NewAppGwIngressController(appGwClient, appGwIdentifier, k8sContext, recorder).Start()
	select {}
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

func getAzAuth(vars environment.EnvVariables) (autorest.Authorizer, error) {
	if vars.AuthLocation == "" {
		// requires aad-pod-identity to be deployed in the AKS cluster
		// see https://github.com/Azure/aad-pod-identity for more information
		glog.V(1).Infoln("Creating authorizer from Azure Managed Service Identity")
		return auth.NewAuthorizerFromEnvironment()
	}
	glog.V(1).Infoln("Creating authorizer from file referenced by AZURE_AUTH_LOCATION")
	return auth.NewAuthorizerFromFile(network.DefaultBaseURI)
}

func waitForAzureAuth(envVars environment.EnvVariables, client network.ApplicationGatewaysClient) {
	maxRetry := 10
	const retryTime = 10 * time.Second
	for counter := 0; counter <= maxRetry; counter++ {
		if _, err := client.Get(context.Background(), envVars.ResourceGroupName, envVars.AppGwName); err != nil {
			glog.Error("Error getting Application Gateway", envVars.AppGwName, err)
			glog.Infof("Retrying in %v", retryTime)
			time.Sleep(retryTime)
		}
		return
	}
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
	eventBroadcaster.StartLogging(glog.Infof)
	eventBroadcaster.StartRecordingToSink(
		&typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")})
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
