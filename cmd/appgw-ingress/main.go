// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import (
	"context"
	go_flag "flag"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/golang/glog"
	flag "github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controller"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
)

var (
	flags = flag.NewFlagSet(`appgw-ingress`, flag.ExitOnError)

	inCluster = flags.Bool("in-cluster", true,
		"If running in a Kubernetes cluster, use the pod secrets for creating a Kubernetes client. Optional.")

	apiServerHost = flags.String("apiserver-host", "",
		"The address of the Kubernetes apiserver. Optional if running in cluster; if omitted, local discovery is attempted.")

	kubeConfigFile = flags.String("kubeconfig", "",
		"Path to kubeconfig file with authorization and master location information.")

	resyncPeriod = flags.Duration("sync-period", 30*time.Second,
		"Interval at which to re-list and confirm cloud resources.")
)

func main() {
	if err := flags.Parse(os.Args); err != nil {
		glog.Fatalf("Error parsing command line arguments: %v", err.Error())
	}

	// Workaround for "ERROR: logging before flag.Parse"
	// See: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	_ = go_flag.CommandLine.Parse([]string{})

	envVariables := newEnvVariables()
	glog.Infof("Environment Variables:\n%+v", envVariables)
	if len(envVariables.SubscriptionID) == 0 || len(envVariables.ResourceGroupName) == 0 || len(envVariables.AppGwName) == 0 || len(envVariables.WatchNamespace) == 0 {
		glog.Fatalf("Error while initializing values from environment. Please check helm configuration for missing values.")
	}

	setLoggingOptions()

	kubeClient := kubeClient()

	fileLocation := envVariables.AuthLocation

	var err error
	var azureAuth autorest.Authorizer

	if fileLocation == "" {
		// requires aad-pod-identity to be deployed in the AKS cluster
		// see https://github.com/Azure/aad-pod-identity for more information
		glog.V(1).Infoln("Creating authorizer from MSI")
		azureAuth, err = auth.NewAuthorizerFromEnvironment()
	} else {
		glog.V(1).Infoln("Creating authorizer from file referenced by AZURE_AUTH_LOCATION")
		azureAuth, err = auth.NewAuthorizerFromFile(network.DefaultBaseURI)
	}

	if err != nil || azureAuth == nil {
		glog.Fatalf("Error creating Azure client from config: %v", err.Error())
	}

	appGwIdentifier := appgw.Identifier{
		SubscriptionID: envVariables.SubscriptionID,
		ResourceGroup:  envVariables.ResourceGroupName,
		AppGwName:      envVariables.AppGwName,
	}

	appGwClient := network.NewApplicationGatewaysClient(appGwIdentifier.SubscriptionID)
	appGwClient.Authorizer = azureAuth

	// wait until azureAuth becomes valid
	for true {
		ctx := context.Background()
		_, err := appGwClient.Get(ctx, appGwIdentifier.ResourceGroup, appGwIdentifier.AppGwName)
		if err == nil {
			break
		} else {
			glog.Errorf("unable to get specified ApplicationGateway [%v], error=[%v]", appGwIdentifier.AppGwName, err.Error())
		}
		retryTime := 10 * time.Second
		glog.Infof("Retrying in %v", retryTime.String())
		time.Sleep(retryTime)
	}

	namespace := envVariables.WatchNamespace

	if len(namespace) == 0 {
		glog.Fatal("Error creating informers, namespace is not specified")
	}
	_, err = kubeClient.CoreV1().Namespaces().Get(namespace, metav1.GetOptions{})
	if err != nil {
		glog.Fatalf("Error creating informers, namespace [%v] is not found: %v", namespace, err.Error())
	}

	ctx := k8scontext.NewContext(kubeClient, namespace, *resyncPeriod)
	appGwController := controller.NewAppGwIngressController(appGwClient, appGwIdentifier, ctx)

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
		glog.Fatalf("Failed to create client: %v", err.Error())
	}

	return kubeClient
}

func getKubeClientConfig() *rest.Config {
	if *inCluster {
		config, err := rest.InClusterConfig()
		if err != nil {
			glog.Fatalf("Error creating client configuration: %v", err.Error())
		}
		return config
	}

	if *apiServerHost == "" {
		glog.Fatalf("when not running in a cluster you must specify --apiserver-host")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfigFile)

	if err != nil {
		glog.Fatalf("error creating client configuration: %v", err.Error())
	}

	return config
}
