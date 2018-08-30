package main

import (
	go_flag "flag"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-06-01/network"

	"github.com/Azure/Networking-AppGW-k8s/pkg/appgw"
	"github.com/Azure/Networking-AppGW-k8s/pkg/controller"
	"github.com/Azure/Networking-AppGW-k8s/pkg/k8scontext"

	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"

	"github.com/golang/glog"
	flag "github.com/spf13/pflag"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

var (
	flags = flag.NewFlagSet(`appgw-ingress`, flag.ExitOnError)

	inCluster = flags.Bool("in-cluster", true,
		"If running in a Kubernetes cluster, use the pod secrets for creating a Kubernetes client. Optional.")

	apiServerHost = flags.String("apiserver-host", "",
		"The address of the Kubernetes apiserver. Optional if running in cluster; if omitted, local discovery is attempted.")

	kubeConfigFile = flags.String("kubeconfig", "",
		"Path to kubeconfig file with authorization and master location information.")

	resyncPeriod = flags.Duration("sync-period", 50*time.Second,
		"Interval at which to re-list and confirm cloud resources.")
)

func main() {
	flags.Parse(os.Args)

	setLoggingOptions()

	kubeClient := kubeClient()
	azureAuth := azAuthFromConfigMap(kubeClient, "azure-config")
	// azureConfig := azureConfigFromConfigMap(kubeClient, "azure-config")

	appGwIdentifier := appgw.NewIdentifierFromEnv()

	appGwClient := network.NewApplicationGatewaysClient(appGwIdentifier.SubscriptionID)
	appGwClient.Authorizer = azureAuth

	ctx := k8scontext.NewContext(kubeClient, "default", *resyncPeriod)
	appGwController := controller.NewAppGwIngressController(kubeClient, appGwClient, appGwIdentifier, ctx)

	go appGwController.Start()

	for true {
		time.Sleep(1 * time.Minute)
	}
}

func setLoggingOptions() {
	go_flag.Lookup("logtostderr").Value.Set("true")
	go_flag.Set("v", "3")
}

func kubeClient() kubernetes.Interface {
	config := getKubeClientConfig()

	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		glog.Fatalf("Failed to create client: %v", err)
	}

	return kubeClient
}

func getKubeClientConfig() *rest.Config {
	if *inCluster {
		config, err := rest.InClusterConfig()
		if err != nil {
			glog.Fatalf("Error creating client configuration: %v", err)
		}
		return config
	}

	if *apiServerHost == "" {
		glog.Fatalf("when not running in a cluster you must specify --apiserver-host")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags(*apiServerHost, *kubeConfigFile)

	if err != nil {
		glog.Fatalf("error creating client configuration: %v", err)
	}

	return config
}

func azAuthFromConfigMap(kubeclient kubernetes.Interface, mapName string) autorest.Authorizer {
	authn, err := auth.NewAuthorizerFromFile(network.DefaultBaseURI)
	if err != nil {
		glog.Fatalf("Error creating Azure client from config: %v", err)
	}

	return authn
}
