// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import (
	"flag"
	"os"
	"os/signal"
	"sort"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Azure/go-autorest/autorest"
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
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controller"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned"
	istio "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/httpserver"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/version"
)

const (
	verbosityFlag     = "verbosity"
	maxAuthRetryCount = 10
	retryPause        = 10 * time.Second
	resyncPause       = 30 * time.Second
)

var (
	flags          = pflag.NewFlagSet(`appgw-ingress`, pflag.ExitOnError)
	inCluster      = flags.Bool("in-cluster", true, "If running in a Kubernetes cluster, use the pod secrets for creating a Kubernetes client. Optional.")
	apiServerHost  = flags.String("apiserver-host", "", "The address of the Kubernetes API Server. Optional if running in cluster; if omitted, local discovery is attempted.")
	kubeConfigFile = flags.String("kubeconfig", "", "Path to kubeconfig file with authorization and master location information.")
	resyncPeriod   = flags.Duration("sync-period", resyncPause, "Interval at which to re-list and confirm cloud resources.")
	versionInfo    = flags.Bool("version", false, "Print version")
	verbosity      = flags.Int(verbosityFlag, 1, "Set logging verbosity level")
)

func main() {
	// Log output is buffered... Calling Flush before exiting guarantees all log output is written.
	defer glog.Flush()
	if err := flags.Parse(os.Args); err != nil {
		glog.Fatal("Error parsing command line arguments:", err)
	}

	env := environment.GetEnv()

	verbosity = to.IntPtr(getVerbosity(*verbosity, env.VerbosityLevel))
	if *versionInfo {
		version.PrintVersionAndExit()
	}

	// Workaround for "ERROR: logging before flag.Parse"
	// See: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	_ = flag.CommandLine.Parse([]string{})
	_ = flag.Lookup("logtostderr").Value.Set("true")
	_ = flag.Set("v", strconv.Itoa(*verbosity))

	apiConfig := getKubeClientConfig()
	kubeClient := kubernetes.NewForConfigOrDie(apiConfig)
	crdClient := versioned.NewForConfigOrDie(apiConfig)
	istioCrdClient := istio.NewForConfigOrDie(apiConfig)
	recorder := getEventRecorder(kubeClient)
	namespaces := getNamespacesToWatch(env.WatchNamespace)
	k8sContext := k8scontext.NewContext(kubeClient, crdClient, istioCrdClient, namespaces, *resyncPeriod)
	agicPod := k8sContext.GetAGICPod(env)
	metricStore := metricstore.NewMetricStore(env)

	if env.AppGwName == "" {
		env.AppGwName = env.ReleaseName
	}

	if infraSubID, infraResourceGp, err := k8sContext.GetInfrastructureResourceGroupID(); env.SubscriptionID == "" && err != nil {
		env.SubscriptionID = string(infraSubID)
		if env.ResourceGroupName == "" {
			env.ResourceGroupName = string(infraResourceGp)
		}
	}

	if err := environment.ValidateEnv(env); err != nil {
		glog.Fatal("Error while initializing values from environment. Please check helm configuration for missing values: ", err)
	}

	glog.V(3).Infof("App Gateway Details: Subscription: %s, Resource Group: %s, Name: %s", env.SubscriptionID, env.ResourceGroupName, env.AppGwName)

	var err error
	var authorizer autorest.Authorizer
	if authorizer, err = azure.GetAuthorizerWithRetry(env.AuthLocation, maxAuthRetryCount, retryPause); err != nil {
		glog.Fatal("Failed obtaining authentication token for Azure Resource Manager")
	}

	azClient := azure.NewAzClient(azure.SubscriptionID(env.SubscriptionID), azure.ResourceGroup(env.ResourceGroupName), azure.ResourceName(env.AppGwName), authorizer)
	if err = azure.WaitForAzureAuth(azClient, maxAuthRetryCount, retryPause); err != nil {
		if err == azure.ErrAppGatewayNotFound && env.EnableDeployAppGateway {
			err = azClient.DeployGateway(env.AppGwSubnetID)
			if err != nil {
				glog.Fatal("Failed in deploying App gateway", err)
			}
		} else {
			glog.Fatal("Failed authenticating with Azure Resource Manager: ", err)
		}
	}

	appGwIdentifier := appgw.Identifier{
		SubscriptionID: env.SubscriptionID,
		ResourceGroup:  env.ResourceGroupName,
		AppGwName:      env.AppGwName,
	}

	// namespace validations
	if err := validateNamespaces(namespaces, kubeClient); err != nil {
		glog.Fatal(err) // side-effect: will panic on non-existent namespace
	}
	if len(namespaces) == 0 {
		glog.Info("Ingress Controller will observe all namespaces.")
	} else {
		glog.Info("Ingress Controller will observe the following namespaces:", strings.Join(namespaces, ","))
	}

	// fatal config validations
	appGw, _ := azClient.GetGateway()
	if err := appgw.FatalValidateOnExistingConfig(recorder, appGw.ApplicationGatewayPropertiesFormat, env); err != nil {
		glog.Fatal("Got a fatal validation error on existing Application Gateway config. Please update Application Gateway or the controller's helm config. Error:", err)
	}

	appGwIngressController := controller.NewAppGwIngressController(azClient, appGwIdentifier, k8sContext, recorder, metricStore, agicPod)

	if err := appGwIngressController.Start(env); err != nil {
		glog.Fatal("Could not start AGIC: ", err)
	}

	httpServer := httpserver.NewHTTPServer(
		appGwIngressController,
		metricStore,
		env.HTTPServicePort)
	httpServer.Start()

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	appGwIngressController.Stop()
	httpServer.Stop()
	glog.Info("Goodbye!")
}

func validateNamespaces(namespaces []string, kubeClient *kubernetes.Clientset) error {
	var nonExistent []string
	for _, ns := range namespaces {
		if _, err := kubeClient.CoreV1().Namespaces().Get(ns, metav1.GetOptions{}); err != nil {
			nonExistent = append(nonExistent, ns)
		}
	}
	if len(nonExistent) > 0 {
		glog.Errorf("Error creating informers; Namespaces do not exist or Ingress Controller has no access to: %v", strings.Join(nonExistent, ","))
		return ErrNoSuchNamespace
	}
	return nil
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
