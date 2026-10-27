// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import (
	"context"
	"flag"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2021-03-01/network"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/spf13/pflag"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	ctrl_client "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/appgw"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/azure"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/cni"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controller"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned"
	multicluster "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/azure_multicluster_crd_client/clientset/versioned"
	istio "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client/clientset/versioned"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/events"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/httpserver"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8s"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/k8scontext"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/metricstore"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/utils"
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
	cleanupOEC     = flags.Bool("cleanup-oec", false, "Cleanup OverlayExtensionConfig resources")
)

var allowedSkus = map[n.ApplicationGatewayTier]interface{}{
	n.ApplicationGatewayTierStandardV2: nil,
	n.ApplicationGatewayTierWAFV2:      nil,
}

func main() {
	// Log output is buffered... Calling Flush before exiting guarantees all log output is written.
	klog.InitFlags(nil)
	defer klog.Flush()
	if err := flags.Parse(os.Args); err != nil {
		klog.Fatal("Error parsing command line arguments:", err)
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
		klog.Infof("Unable to load cloud provider config '%s'. Error: %s", env.CloudProviderConfigLocation, err.Error())
	}

	env.Consolidate(cpConfig)

	// Workaround for "ERROR: logging before flag.Parse"
	// See: https://github.com/kubernetes/kubernetes/issues/17162#issuecomment-225596212
	_ = flag.CommandLine.Parse([]string{})
	_ = flag.Lookup("logtostderr").Value.Set("true")
	_ = flag.Set("v", strconv.Itoa(*verbosity))

	apiConfig := getKubeClientConfig()
	scheme, err := k8s.NewScheme()
	if err != nil {
		klog.Fatalf("Failed to create k8s scheme: %v", err)
	}

	ctrlClient, err := ctrl_client.New(apiConfig, ctrl_client.Options{
		Scheme: scheme,
	})
	if err != nil {
		klog.Fatalf("Failed to create controller-runtime client: %v", err)
	}

	if *cleanupOEC {
		if err := cni.CleanupOverlayExtensionConfigs(ctrlClient, env.AGICPodNamespace, env.AddonMode); err != nil {
			klog.Fatalf("Failed to cleanup OverlayExtensionConfig resources: %v", err)
		}
		klog.Info("Successfully cleaned up OverlayExtensionConfig resources")
		return
	}

	kubeClient := kubernetes.NewForConfigOrDie(apiConfig)
	k8scontext.IsNetworkingV1PackageSupported = k8scontext.SupportsNetworkingPackage(kubeClient)
	k8scontext.IsInMultiClusterMode = env.MultiClusterMode
	crdClient := versioned.NewForConfigOrDie(apiConfig)
	istioCrdClient := istio.NewForConfigOrDie(apiConfig)
	multiClusterCrdClient := multicluster.NewForConfigOrDie(apiConfig)
	recorder := getEventRecorder(kubeClient, env.IngressClassControllerName)
	namespaces := getNamespacesToWatch(env.WatchNamespace)
	metricStore := metricstore.NewMetricStore(env)
	metricStore.Start()
	k8sContext := k8scontext.NewContext(kubeClient, crdClient, multiClusterCrdClient, istioCrdClient, namespaces, *resyncPeriod, metricStore, env)
	agicPod := k8sContext.GetAGICPod(env)

	if err := environment.ValidateEnv(env); err != nil {
		errorLine := fmt.Sprint("Error while initializing values from environment. Please check helm configuration for missing values: ", err)
		if agicPod != nil {
			recorder.Event(agicPod, v1.EventTypeWarning, events.ReasonValidatonError, errorLine)
		}
		klog.Fatal(errorLine)
	}

	uniqueUserAgentSuffix := utils.RandStringRunes(10)
	if agicPod != nil {
		uniqueUserAgentSuffix = agicPod.Name
	}
	klog.Infof("Using User Agent Suffix='%s' when communicating with ARM", uniqueUserAgentSuffix)

	azClient := azure.NewAzClient(azure.SubscriptionID(env.SubscriptionID), azure.ResourceGroup(env.ResourceGroupName), azure.ResourceName(env.AppGwName), uniqueUserAgentSuffix, env.ClientID)
	appGwIdentifier := appgw.Identifier{
		SubscriptionID: env.SubscriptionID,
		ResourceGroup:  env.ResourceGroupName,
		AppGwName:      env.AppGwName,
	}

	klog.V(3).Infof("Application Gateway Details: Subscription=\"%s\" Resource Group=\"%s\" Name=\"%s\"", env.SubscriptionID, env.ResourceGroupName, env.AppGwName)

	var authorizer autorest.Authorizer
	if authorizer, err = azure.GetAuthorizerWithRetry(env.AuthLocation, env.UseManagedIdentityForPod, cpConfig, maxRetryCount, retryPause); err != nil {
		errorLine := fmt.Sprint("Failed obtaining authentication token for Azure Resource Manager: ", err)
		if agicPod != nil {
			recorder.Event(agicPod, v1.EventTypeWarning, events.ReasonARMAuthFailure, errorLine)
		}
		klog.Fatal(errorLine)
	} else {
		azClient.SetAuthorizer(authorizer)
	}

	// Check if Application Gateway exists/have get access
	// If AGIC's service principal or managed identity doesn't have read access to the Application Gateway's resource group,
	// then AGIC can't read it's role assignments to look for the needed permission.
	// Instead we perform a simple GET request to check both that the Application Gateway exists as well as implicitly make sure that AGIC has read access to it.
	err = azClient.WaitForGetAccessOnGateway(maxRetryCount)
	if err != nil {
		if controllererrors.IsErrorCode(err, controllererrors.ErrorApplicationGatewayNotFound) && env.EnableDeployAppGateway {
			if env.AppGwSubnetID != "" {
				err = azClient.DeployGatewayWithSubnet(env.AppGwSubnetID, env.AppGwSkuName)
			} else if cpConfig != nil {
				err = azClient.DeployGatewayWithVnet(azure.ResourceGroup(cpConfig.VNetResourceGroup), azure.ResourceName(cpConfig.VNetName), azure.ResourceName(env.AppGwSubnetName), env.AppGwSubnetPrefix, env.AppGwSkuName)
			}

			if err != nil {
				errorLine := fmt.Sprint("Failed in deploying Application Gateway", err)
				if agicPod != nil {
					recorder.Event(agicPod, v1.EventTypeWarning, events.ReasonFailedDeployingAppGw, errorLine)
				}
				klog.Fatal(errorLine)
			}
		} else {
			errorLine := fmt.Sprint("Failed getting Application Gateway: ", err)
			if agicPod != nil {
				recorder.Event(agicPod, v1.EventTypeWarning, events.ReasonARMAuthFailure, errorLine)
			}
			klog.Fatal(errorLine)
		}
	}

	// namespace validations
	if err := validateNamespaces(namespaces, kubeClient); err != nil {
		klog.Fatal(err) // side-effect: will panic on non-existent namespace
	}
	if len(namespaces) == 0 {
		klog.Info("Ingress Controller will observe all namespaces.")
	} else {
		klog.Info("Ingress Controller will observe the following namespaces:", strings.Join(namespaces, ","))
	}

	// fatal config validations
	appGw, _ := azClient.GetGateway()
	if _, exists := allowedSkus[appGw.Sku.Tier]; !exists {
		errorLine := fmt.Sprintf("App Gateway SKU Tier %s is not supported by AGIC version %s; (v0.10.0 supports App Gwy v1)", appGw.Sku.Tier, appgw.GetVersion())
		if agicPod != nil {
			recorder.Event(agicPod, v1.EventTypeWarning, events.UnsupportedAppGatewaySKUTier, errorLine)
		}

		// Slow down the cycling of the AGIC pod.
		time.Sleep(5 * time.Second)
		klog.Fatal(errorLine)
	}

	cniReconciler := cni.NewReconciler(azClient, ctrlClient, recorder, cpConfig, appGw, agicPod, env.AGICPodNamespace, env.AddonMode)

	// create a new agic controller
	appGwIngressController := controller.NewAppGwIngressController(azClient, appGwIdentifier, k8sContext, recorder, metricStore, cniReconciler, agicPod, env.HostedOnUnderlay)

	// initialize the http server and start it
	httpServer := httpserver.NewHTTPServer(
		appGwIngressController,
		metricStore,
		env.HTTPServicePort)
	httpServer.Start()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go runWithLeaderElection(ctx, kubeClient, env, func(ctx context.Context) {
		if err := appGwIngressController.Start(env); err != nil {
			errorLine := fmt.Sprint("Could not start AGIC: ", err)
			if agicPod != nil {
				recorder.Event(agicPod, v1.EventTypeWarning, events.ReasonARMAuthFailure, errorLine)
			}
			klog.Fatal(errorLine)
		}
	}, appGwIngressController.Stop)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	httpServer.Stop()
	klog.Info("Goodbye!")
}

func runWithLeaderElection(ctx context.Context, kubeClient *kubernetes.Clientset, env environment.EnvVariables, start func(ctx context.Context), stop func()) {

	id, err := os.Hostname()
	if err != nil {
		klog.Fatalf("Error getting hostname: %v", err)
	}

	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      env.IngressClassControllerName + "-lease",
			Namespace: env.AGICPodNamespace,
		},
		Client: kubeClient.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}

	leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
		Lock:            lock,
		ReleaseOnCancel: true,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				klog.Infof("Became leader: %s", id)
				start(ctx)
			},
			OnStoppedLeading: func() {
				klog.Infof("Leader lost: %s", id)
				stop()
			},
			OnNewLeader: func(identity string) {
				if identity != id {
					klog.Infof("New leader elected: %s", identity)
				}
			},
		},
	})
}
