// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package runner

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

const (
	// KubeConfigEnvVar is the environment variable for KUBECONFIG.
	KubeConfigEnvVar = "KUBECONFIG"
)

func getClient() (*kubernetes.Clientset, error) {
	var kubeConfig *rest.Config
	var err error
	kubeConfigFile := os.Getenv(KubeConfigEnvVar)
	if kubeConfigFile == "" {
		return nil, fmt.Errorf("KUBECONFIG is not set")
	}

	kubeConfig, err = clientcmd.BuildConfigFromFlags("", kubeConfigFile)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	return clientset, nil
}

func parseK8sYaml(fileName string) ([]runtime.Object, error) {
	fileR, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	acceptedK8sTypes := regexp.MustCompile(`(Namespace|Deployment|Service|Ingress|Secret|ConfigMap)`)
	fileAsString := string(fileR[:])
	sepYamlfiles := strings.Split(fileAsString, "---")
	retVal := make([]runtime.Object, 0, len(sepYamlfiles))
	for _, f := range sepYamlfiles {
		if f == "\n" || f == "" {
			// ignore empty cases
			continue
		}

		obj, groupVersionKind, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(f), nil, nil)
		if err != nil {
			return nil, err
		}

		if !acceptedK8sTypes.MatchString(groupVersionKind.Kind) {
			klog.Infof("Skipping object with type: %s", groupVersionKind.Kind)
		} else {
			retVal = append(retVal, obj)
		}

	}
	return retVal, nil
}

func applyYaml(clientset *kubernetes.Clientset, namespaceName string, fileName string) error {
	// create objects in the yaml
	fileObjects, err := parseK8sYaml(fileName)
	if err != nil {
		return err
	}

	for _, objs := range fileObjects {
		if secret, ok := objs.(*v1.Secret); ok {
			if _, err := clientset.CoreV1().Secrets(namespaceName).Create(secret); err != nil {
				return err
			}
		}
		if ingress, ok := objs.(*v1beta1.Ingress); ok {
			if _, err := clientset.ExtensionsV1beta1().Ingresses(namespaceName).Create(ingress); err != nil {
				return err
			}
		}
		if service, ok := objs.(*v1.Service); ok {
			if _, err := clientset.CoreV1().Services(namespaceName).Create(service); err != nil {
				return err
			}
		}
		if deployment, ok := objs.(*appsv1.Deployment); ok {
			if _, err := clientset.AppsV1().Deployments(namespaceName).Create(deployment); err != nil {
				return err
			}
		}
		if cm, ok := objs.(*v1.ConfigMap); ok {
			if _, err := clientset.CoreV1().ConfigMaps(namespaceName).Create(cm); err != nil {
				return err
			}
		}
	}
	return nil
}

func cleanUp(clientset *kubernetes.Clientset) error {
	namespaces, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		return err
	}

	var namespacesToDelete []v1.Namespace
	for _, ns := range namespaces.Items {
		if strings.HasPrefix(ns.Name, "e2e-") {
			namespacesToDelete = append(namespacesToDelete, ns)
		}
	}

	if len(namespacesToDelete) == 0 {
		return nil
	}

	deleteOptions := &metav1.DeleteOptions{
		GracePeriodSeconds: to.Int64Ptr(0),
	}

	klog.Infof("Deleting namespaces [%+v]...", namespacesToDelete)
	for _, ns := range namespacesToDelete {
		if err = clientset.CoreV1().Namespaces().Delete(ns.Name, deleteOptions); err != nil {
			return err
		}
	}

	klog.Info("Waiting for namespace to get deleted...")
	for _, ns := range namespacesToDelete {
		for i := 1; i <= 100; i++ {
			_, err = clientset.CoreV1().Namespaces().Get(ns.Name, metav1.GetOptions{})
			if err != nil {
				break
			}

			klog.Warning("Trying again...", i)
			time.Sleep(time.Second)
		}
	}

	return nil
}

func getPublicIP(clientset *kubernetes.Clientset, namespaceName string) (string, error) {
	for i := 1; i <= 100; i++ {
		ingresses, err := clientset.ExtensionsV1beta1().Ingresses(namespaceName).List(metav1.ListOptions{})
		if err != nil {
			return "", err
		}

		if ingresses == nil || len((*ingresses).Items) == 0 {
			return "", fmt.Errorf("Unable to find ingress in namespace %s", namespaceName)
		}

		ingress := (*ingresses).Items[0]
		if len(ingress.Status.LoadBalancer.Ingress) == 0 {
			klog.Warning("Trying again...", i)
			time.Sleep(time.Second)
			continue
		}

		publicIP := ingress.Status.LoadBalancer.Ingress[0].IP
		if publicIP != "" {
			return publicIP, nil
		}

		klog.Warning("Trying again...", i)
		time.Sleep(time.Second)
	}

	return "", fmt.Errorf("Timed out while finding ingress IP in namespace %s", namespaceName)
}

func makeGetRequest(url string, host string, statusCode int, inSecure bool) (*http.Response, error) {
	var resp *http.Response
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	if len(host) > 0 {
		req.Host = host
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: inSecure},
	}

	client := &http.Client{
		Transport: tr,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	client.Timeout = 2 * time.Second

	klog.Warning("Sending GET request...")
	for i := 1; i <= 100; i++ {
		resp, err := client.Do(req)
		if err != nil {
			klog.Warningf("Trying again %d. Sleeping for 5 seconds. Err: %+v...", i, err)
			time.Sleep(5 * time.Second)
			continue
		}

		if resp.StatusCode == statusCode {
			klog.Infof("Got expected status code %d with url '%s' with host '%s'. Response: [%+v]", statusCode, url, host, resp)
			return resp, nil
		}

		klog.Warningf("Trying again %d. Sleeping for 5 seconds. Got response [%+v].", i, resp)
		time.Sleep(5 * time.Second)
	}

	return nil, fmt.Errorf("Unable to get status code %d with url '%s' with host '%s'. Response: [%+v]", statusCode, url, host, resp)
}
