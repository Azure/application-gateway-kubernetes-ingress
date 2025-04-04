// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package main

import (
	"context"
	"os"
	"sort"
	"strconv"
	"strings"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"
	"k8s.io/klog/v2"

	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/controllererrors"
	agiccrdscheme "github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client/clientset/versioned/scheme"
	"github.com/Azure/application-gateway-kubernetes-ingress/pkg/environment"
)

func validateNamespaces(namespaces []string, kubeClient *kubernetes.Clientset) error {
	var nonExistent []string
	for _, ns := range namespaces {
		if _, err := kubeClient.CoreV1().Namespaces().Get(context.TODO(), ns, metav1.GetOptions{}); err != nil {
			nonExistent = append(nonExistent, ns)
		}
	}
	if len(nonExistent) > 0 {
		err := controllererrors.NewErrorf(
			controllererrors.ErrorNoSuchNamespace,
			"error creating informers; Namespaces do not exist or Ingress Controller has no access to: %v", strings.Join(nonExistent, ","),
		)
		klog.Error(err.Error())
		return err
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
			klog.Fatal("Error creating in-cluster client configuration:", err)
		}
		return config
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", *kubeConfigFile)
	if err != nil {
		klog.Fatal("error creating client configuration:", err)
	}

	return config
}

func getEventRecorder(kubeClient kubernetes.Interface, ingressClassControllerName string) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	sink := &typedcorev1.EventSinkImpl{Interface: kubeClient.CoreV1().Events("")}
	eventBroadcaster.StartRecordingToSink(sink)
	hostname, err := os.Hostname()
	if err != nil {
		klog.Error("Could not obtain host name from the operating system", err)
		hostname = "unknown-hostname"
	}
	source := v1.EventSource{
		Component: ingressClassControllerName,
		Host:      hostname,
	}

	s := scheme.Scheme
	agiccrdscheme.AddToScheme(s)
	return eventBroadcaster.NewRecorder(scheme.Scheme, source)
}

func getVerbosity(flagVerbosity int, envVerbosity string) int {
	envVerbosityInt, err := strconv.Atoi(envVerbosity)
	if err != nil {
		klog.Infof("Using verbosity level %d from CLI flag %s", flagVerbosity, verbosityFlag)
		return flagVerbosity
	}
	klog.Infof("Using verbosity level %d from environment variable %s", envVerbosityInt, environment.VerbosityLevelVarName)
	return envVerbosityInt
}
