// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import (
	"context"
	"flag"
	"os"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/network/mgmt/2018-12-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controller"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
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
)

func main() {
	// Log output is buffered... Calling Flush before exiting guarantees all log output is written.
	defer glog.Flush()
	if err := flags.Parse(os.Args); err != nil {
		glog.Fatal("Error parsing command line arguments:", err)
	}

	// Workaround for "ERROR: logging before flag.Parse"
	// See: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	_ = flag.CommandLine.Parse([]string{})
	_ = flag.Lookup("logtostderr").Value.Set("true")
	_ = flag.Set("v", "3")

	env := getEnvVars()

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
	ctx := k8scontext.NewContext(getKubeClient(env), env.WatchNamespace, *resyncPeriod)
	go controller.NewAppGwIngressController(appGwClient, appGwIdentifier, ctx).Start()
	select {}
}

func getKubeClient(env envVariables) *kubernetes.Clientset {
	kubeClient, err := kubernetes.NewForConfig(getKubeClientConfig())
	if err != nil {
		glog.Fatal("Error creating Kubernetes client: ", err)
	}
	if _, err = kubeClient.CoreV1().Namespaces().Get(env.WatchNamespace, metav1.GetOptions{}); err != nil {
		glog.Fatalf("Error creating informers, namespace [%v] is not found: %v", env.WatchNamespace, err.Error())
	}
	return kubeClient
}

func getAzAuth(vars envVariables) (autorest.Authorizer, error) {
	if vars.AuthLocation == "" {
		// requires aad-pod-identity to be deployed in the AKS cluster
		// see https://github.com/Azure/aad-pod-identity for more information
		glog.V(1).Infoln("Creating authorizer from Azure Managed Service Identity")
		return auth.NewAuthorizerFromEnvironment()
	}
	glog.V(1).Infoln("Creating authorizer from file referenced by AZURE_AUTH_LOCATION")
	return auth.NewAuthorizerFromFile(network.DefaultBaseURI)
}

func waitForAzureAuth(envVars envVariables, client network.ApplicationGatewaysClient) {
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
