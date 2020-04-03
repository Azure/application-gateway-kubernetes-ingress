// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2019-09-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/glog"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controller"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned"
	istio "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/httpserver"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/version"
)

const (
	verbosityFlag = "verbosity"
	maxRetryCount = 10
	retryPause    = 10 * time.Second
	resyncPause   = 30 * time.Second
)

var (
	flags          = pflag.NewFlagSet(`appgw-ingress`, pflag.ExitOnError)
	inCluster      = flags.Bool("in-cluster", true, "If running in a Kubernetes cluster, use the pod secrets for creating a Kubernetes client. Optional.")
	kubeConfigFile = flags.String("kubeconfig", "", "Path to kubeconfig file with authorization and master location information.")
	resyncPeriod   = flags.Duration("sync-period", resyncPause, "Interval at which to re-list and confirm cloud resources.")
	versionInfo    = flags.Bool("version", false, "Print version")
	verbosity      = flags.Int(verbosityFlag, 1, "Set logging verbosity level")
)

var allowedSkus = map[n.ApplicationGatewayTier]interface{}{
	n.ApplicationGatewayTierStandardV2: nil,
	n.ApplicationGatewayTierWAFV2:      nil,
}

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

	// get the details from Cloud Provider Config
	// Reference: https://github.com/kubernetes-sigs/cloud-provider-azure/blob/master/docs/cloud-provider-config.md#cloud-provider-config
	cpConfig, err := azure.NewCloudProviderConfig(env.CloudProviderConfigLocation)
	if err != nil {
		glog.Info("Unable to load cloud provider config:", env.CloudProviderConfigLocation)
	}

	env.Consolidate(cpConfig)

	// Workaround for "ERROR: logging before flag.Parse"
	// See: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	_ = flag.CommandLine.Parse([]string{})
	_ = flag.Lookup("logtostderr").Value.Set("true")
	_ = flag.Set("v", strconv.Itoa(*verbosity))

	apiConfig := getKubeClientConfig()
	kubeClient := kubernetes.NewForConfigOrDie(apiConfig)
	k8scontext.IsNetworkingV1Beta1PackageSupported, _ = k8scontext.SupportsNetworkingPackage(kubeClient)
	crdClient := versioned.NewForConfigOrDie(apiConfig)
	istioCrdClient := istio.NewForConfigOrDie(apiConfig)
	recorder := getEventRecorder(kubeClient)
	namespaces := getNamespacesToWatch(env.WatchNamespace)
	metricStore := metricstore.NewMetricStore(env)
	metricStore.Start()
	k8sContext := k8scontext.NewContext(kubeClient, crdClient, istioCrdClient, namespaces, *resyncPeriod, metricStore)
	agicPod := k8sContext.GetAGICPod(env)

	if err := environment.ValidateEnv(env); err != nil {
		errorLine := fmt.Sprint("Error while initializing values from environment. Please check helm configuration for missing values: ", err)
		if agicPod != nil {
			recorder.Event(agicPod, v1.EventTypeWarning, events.ReasonValidatonError, errorLine)
		}
		glog.Fatal(errorLine)
	}

	azClient := azure.NewAzClient(azure.SubscriptionID(env.SubscriptionID), azure.ResourceGroup(env.ResourceGroupName), azure.ResourceName(env.AppGwName))
	appGwIdentifier := appgw.Identifier{
		SubscriptionID: env.SubscriptionID,
		ResourceGroup:  env.ResourceGroupName,
		AppGwName:      env.AppGwName,
	}

	// create a new agic controller
	appGwIngressController := controller.NewAppGwIngressController(azClient, appGwIdentifier, k8sContext, recorder, metricStore, agicPod, env.HostedOnUnderlay)

	// initialize the http server and start it
	httpServer := httpserver.NewHTTPServer(
		appGwIngressController,
		metricStore,
		env.HTTPServicePort)
	httpServer.Start()

	glog.V(3).Infof("App Gateway Details: Subscription: %s, Resource Group: %s, Name: %s", env.SubscriptionID, env.ResourceGroupName, env.AppGwName)

	var authorizer autorest.Authorizer
	if authorizer, err = azure.GetAuthorizerWithRetry(env.AuthLocation, env.UseManagedIdentityForPod, cpConfig, maxRetryCount, retryPause); err != nil {
		errorLine := fmt.Sprint("Failed obtaining authentication token for Azure Resource Manager: ", err)
		if agicPod != nil {
			recorder.Event(agicPod, v1.EventTypeWarning, events.ReasonARMAuthFailure, errorLine)
		}
		glog.Fatal(errorLine)
	} else {
		azClient.SetAuthorizer(authorizer)
	}

	if _, err = azClient.GetGateway(); err != nil {
		if err == azure.ErrAppGatewayNotFound && env.EnableDeployAppGateway {
			if env.AppGwSubnetID != "" {
				err = azClient.DeployGatewayWithSubnet(env.AppGwSubnetID)
			} else if cpConfig != nil {
				err = azClient.DeployGatewayWithVnet(azure.ResourceGroup(cpConfig.VNetResourceGroup), azure.ResourceName(cpConfig.VNetName), azure.ResourceName(env.AppGwSubnetName), env.AppGwSubnetPrefix)
			}

			if err != nil {
				errorLine := fmt.Sprint("Failed in deploying App gateway", err)
				if agicPod != nil {
					recorder.Event(agicPod, v1.EventTypeWarning, events.ReasonFailedDeployingAppGw, errorLine)
				}
				glog.Fatal(errorLine)
			}
		} else {
			errorLine := fmt.Sprint("Failed authenticating with Azure Resource Manager: ", err)
			if agicPod != nil {
				recorder.Event(agicPod, v1.EventTypeWarning, events.ReasonARMAuthFailure, errorLine)
			}
			glog.Fatal(errorLine)
		}
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

	if _, exists := allowedSkus[appGw.Sku.Tier]; !exists {
		errorLine := fmt.Sprintf("App Gateway SKU Tier %s is not supported by AGIC version %s; (v0.10.0 supports App Gwy v1)", appGw.Sku.Tier, appgw.GetVersion())
		if agicPod != nil {
			recorder.Event(agicPod, v1.EventTypeWarning, events.UnsupportedAppGatewaySKUTier, errorLine)
		}
		// Slow down the cycling of the AGIC pod.
		time.Sleep(5 * time.Second)
		glog.Fatal(errorLine)
	}

	// associate route table to application gateway subnet
	if cpConfig != nil && cpConfig.RouteTableName != "" {
		subnetID := *(*appGw.GatewayIPConfigurations)[0].Subnet.ID
		routeTableID := azure.RouteTableID(azure.SubscriptionID(cpConfig.SubscriptionID), azure.ResourceGroup(cpConfig.RouteTableResourceGroup), azure.ResourceName(cpConfig.RouteTableName))

		err = azClient.ApplyRouteTable(subnetID, routeTableID)
		if err != nil {
			glog.V(5).Infof("Unable to associate Application Gateway subnet '%s' with route table '%s' due to error (this is relevant for AKS clusters using 'Kubenet' network plugin): [%+v]",
				subnetID,
				routeTableID,
				err)
		}
	}

	if err := appGwIngressController.Start(env); err != nil {
		errorLine := fmt.Sprint("Could not start AGIC: ", err)
		if agicPod != nil {
			recorder.Event(agicPod, v1.EventTypeWarning, events.ReasonARMAuthFailure, errorLine)
		}
		glog.Fatal(errorLine)
	}

	sigChan := make(chan os.Signal)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	appGwIngressController.Stop()
	httpServer.Stop()
	glog.Info("Goodbye!")
}
