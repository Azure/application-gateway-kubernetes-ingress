// -------------------------------------------------------------------------------------------
// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See License.txt in the project root for license information.
// --------------------------------------------------------------------------------------------

package runner

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	n "github.com/Azure/azure-sdk-for-go/services/network/mgmt/2020-05-01/network"
	a "github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-09-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/google/uuid"
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

	// AGICNamespace is namespace where AGIC is installed
	AGICNamespace = "agic"

	// Contributor is the role defintion ID for the corresponding role in AAD
	Contributor = "b24988ac-6180-42a0-ab88-20f7382dd24c"

	// UserAgent is the user agent used when making Azure requests
	UserAgent = "ingress-appgw-e2e"
)

func getClient() (*kubernetes.Clientset, error) {
	var kubeConfig *rest.Config
	var err error
	kubeConfigFile := GetEnv().KubeConfigFilePath
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

func getApplicationGatewaysClient() (*n.ApplicationGatewaysClient, error) {
	env := GetEnv()

	settings, err := auth.GetSettingsFromEnvironment()
	if err != nil {
		return nil, err
	}

	client := n.NewApplicationGatewaysClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, GetEnv().SubscriptionID)
	var authorizer autorest.Authorizer
	if env.AzureAuthLocation != "" {
		// https://docs.microsoft.com/en-us/azure/developer/go/azure-sdk-authorization#use-file-based-authentication
		authorizer, err = auth.NewAuthorizerFromFile(n.DefaultBaseURI)
	} else {
		authorizer, err = settings.GetAuthorizer()
	}
	if err != nil {
		return nil, err
	}

	client.Authorizer = authorizer
	err = client.AddToUserAgent(UserAgent)
	if err != nil {
		return nil, err
	}

	return &client, nil
}

func getRoleAssignmentsClient() (*a.RoleAssignmentsClient, error) {
	env := GetEnv()

	settings, err := auth.GetSettingsFromEnvironment()
	if err != nil {
		return nil, err
	}

	client := a.NewRoleAssignmentsClientWithBaseURI(settings.Environment.ResourceManagerEndpoint, GetEnv().SubscriptionID)
	var authorizer autorest.Authorizer
	if env.AzureAuthLocation != "" {
		// https://docs.microsoft.com/en-us/azure/developer/go/azure-sdk-authorization#use-file-based-authentication
		authorizer, err = auth.NewAuthorizerFromFile(n.DefaultBaseURI)
	} else {
		authorizer, err = settings.GetAuthorizer()
	}
	if err != nil {
		return nil, err
	}

	client.Authorizer = authorizer
	err = client.AddToUserAgent(UserAgent)
	if err != nil {
		return nil, err
	}

	return &client, nil
}

func addRoleAssignment(roleClient *a.RoleAssignmentsClient, role, scope string) error {
	uuidWithHyphen := uuid.New().String()
	objectID := GetEnv().ObjectID
	klog.Infof("Tring to create role: %s, scope: %s, objectID: %s, name: %s", role, scope, objectID, uuidWithHyphen)
	roleID := fmt.Sprintf("/subscriptions/%s/providers/Microsoft.Authorization/roleDefinitions/%s", GetEnv().SubscriptionID, role)
	assignment, err := roleClient.Create(
		context.TODO(),
		scope,
		uuidWithHyphen,
		a.RoleAssignmentCreateParameters{
			RoleAssignmentProperties: &a.RoleAssignmentProperties{
				PrincipalID:      to.StringPtr(GetEnv().ObjectID),
				RoleDefinitionID: to.StringPtr(roleID),
			},
		})
	if err != nil {
		return err
	}

	klog.Infof("Created role assignment: %s on scope: %s and pricipalId: %s", *assignment.Name, *assignment.Scope, *assignment.PrincipalID)
	return nil
}

func removeRoleAssignments(roleClient *a.RoleAssignmentsClient) error {
	page, err := roleClient.ListForScope(context.TODO(), GetEnv().GetApplicationGatewayResourceID(), "")
	if err != nil {
		return err
	}

	klog.Infof("Got role assignments [%+v]", page)

	if page.Response().Value != nil {
		roleAssignmentList := (*page.Response().Value)
		objectID := GetEnv().ObjectID
		for _, assignment := range roleAssignmentList {
			if strings.EqualFold(*assignment.PrincipalID, objectID) {
				klog.Infof("Deleting role assignment: %s on scope: %s and pricipalId: %s", *assignment.Name, *assignment.Scope, *assignment.PrincipalID)
				_, err := roleClient.Delete(context.TODO(), *assignment.Scope, *assignment.Name)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func deleteAGICPod(clientset *kubernetes.Clientset) error {
	// k delete -n agic pods -l app=ingress-azure
	return clientset.CoreV1().Pods(AGICNamespace).DeleteCollection(
		&metav1.DeleteOptions{},
		metav1.ListOptions{
			LabelSelector: "app=ingress-azure",
		})
}

func deleteAADPodIdentityPods(clientset *kubernetes.Clientset) error {
	// k delete -n default pods -l app=mic
	err := clientset.CoreV1().Pods("default").DeleteCollection(
		&metav1.DeleteOptions{},
		metav1.ListOptions{
			LabelSelector: "app=mic",
		})
	if err != nil {
		return err
	}

	// k delete -n default pods -l component=nmi
	err = clientset.CoreV1().Pods("default").DeleteCollection(
		&metav1.DeleteOptions{},
		metav1.ListOptions{
			LabelSelector: "component=nmi",
		})
	return err
}

func parseK8sYaml(fileName string) ([]runtime.Object, error) {
	fileR, err := ioutil.ReadFile(fileName)
	if err != nil {
		return nil, err
	}

	acceptedK8sTypes := regexp.MustCompile(`(Namespace|Deployment|Service|Ingress|Secret|ConfigMap|Pod)`)
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

func updateYaml(clientset *kubernetes.Clientset, namespaceName string, fileName string) error {
	// create objects in the yaml
	fileObjects, err := parseK8sYaml(fileName)
	if err != nil {
		return err
	}

	for _, objs := range fileObjects {
		if secret, ok := objs.(*v1.Secret); ok {
			nm := secret.Namespace
			if len(nm) == 0 && len(namespaceName) != 0 {
				if _, err := clientset.CoreV1().Secrets(namespaceName).Update(secret); err != nil {
					return err
				}
			} else if len(nm) != 0 {
				if _, err := clientset.CoreV1().Secrets(nm).Update(secret); err != nil {
					return err
				}
			} else {
				return errors.New("namespace is not defined for secrets when update")
			}
		} else if ingress, ok := objs.(*v1beta1.Ingress); ok {
			nm := ingress.Namespace
			if len(nm) == 0 && len(namespaceName) != 0 {
				if _, err := clientset.ExtensionsV1beta1().Ingresses(namespaceName).Update(ingress); err != nil {
					return err
				}
			} else if len(nm) != 0 {
				if _, err := clientset.ExtensionsV1beta1().Ingresses(nm).Update(ingress); err != nil {
					return err
				}
			} else {
				return errors.New("namespace is not defined for ingress when update")
			}
		} else if service, ok := objs.(*v1.Service); ok {
			nm := service.Namespace
			if len(nm) == 0 && len(namespaceName) != 0 {
				if _, err := clientset.CoreV1().Services(namespaceName).Update(service); err != nil {
					return err
				}
			} else if len(nm) != 0 {
				if _, err := clientset.CoreV1().Services(nm).Update(service); err != nil {
					return err
				}
			} else {
				return errors.New("namespace is not defined for service when update")
			}
		} else if deployment, ok := objs.(*appsv1.Deployment); ok {
			nm := deployment.Namespace
			if len(nm) == 0 && len(namespaceName) != 0 {
				if _, err := clientset.AppsV1().Deployments(namespaceName).Update(deployment); err != nil {
					return err
				}
			} else if len(nm) != 0 {
				if _, err := clientset.AppsV1().Deployments(nm).Update(deployment); err != nil {
					return err
				}
			} else {
				return errors.New("namespace is not defined for deployment when update")
			}
		} else if cm, ok := objs.(*v1.ConfigMap); ok {
			nm := cm.Namespace
			if len(nm) == 0 && len(namespaceName) != 0 {
				if _, err := clientset.CoreV1().ConfigMaps(namespaceName).Update(cm); err != nil {
					return err
				}
			} else if len(nm) != 0 {
				if _, err := clientset.CoreV1().ConfigMaps(nm).Update(cm); err != nil {
					return err
				}
			} else {
				return errors.New("namespace is not defined for configmaps when update")
			}
		} else if pod, ok := objs.(*v1.Pod); ok {
			nm := pod.Namespace
			if len(nm) == 0 && len(namespaceName) != 0 {
				if _, err := clientset.CoreV1().Pods(namespaceName).Update(pod); err != nil {
					return err
				}
			} else if len(nm) != 0 {
				if _, err := clientset.CoreV1().Pods(nm).Update(pod); err != nil {
					return err
				}
			} else {
				return errors.New("namespace is not defined for pods when update")
			}
		} else {
			return fmt.Errorf("unable to update YAML. Unknown object type: %v", objs)
		}
	}
	return nil
}

func applyYaml(clientset *kubernetes.Clientset, namespaceName string, fileName string) error {
	// create objects in the yaml
	fileObjects, err := parseK8sYaml(fileName)
	if err != nil {
		return err
	}

	for _, objs := range fileObjects {
		if secret, ok := objs.(*v1.Secret); ok {
			nm := secret.Namespace
			if len(nm) == 0 && len(namespaceName) != 0 {
				if _, err := clientset.CoreV1().Secrets(namespaceName).Create(secret); err != nil {
					return err
				}
			} else if len(nm) != 0 {
				if _, err := clientset.CoreV1().Secrets(nm).Create(secret); err != nil {
					return err
				}
			} else {
				return errors.New("namespace is not defined for secrets when create")
			}
		} else if ingress, ok := objs.(*v1beta1.Ingress); ok {
			nm := ingress.Namespace
			if len(nm) == 0 && len(namespaceName) != 0 {
				if _, err := clientset.ExtensionsV1beta1().Ingresses(namespaceName).Create(ingress); err != nil {
					return err
				}
			} else if len(nm) != 0 {
				if _, err := clientset.ExtensionsV1beta1().Ingresses(nm).Create(ingress); err != nil {
					return err
				}
			} else {
				return errors.New("namespace is not defined for ingress when create")
			}
		} else if service, ok := objs.(*v1.Service); ok {
			nm := service.Namespace
			if len(nm) == 0 && len(namespaceName) != 0 {
				if _, err := clientset.CoreV1().Services(namespaceName).Create(service); err != nil {
					return err
				}
			} else if len(nm) != 0 {
				if _, err := clientset.CoreV1().Services(nm).Create(service); err != nil {
					return err
				}
			} else {
				return errors.New("namespace is not defined for service when create")
			}
		} else if deployment, ok := objs.(*appsv1.Deployment); ok {
			nm := deployment.Namespace
			if len(nm) == 0 && len(namespaceName) != 0 {
				if _, err := clientset.AppsV1().Deployments(namespaceName).Create(deployment); err != nil {
					return err
				}
			} else if len(nm) != 0 {
				if _, err := clientset.AppsV1().Deployments(nm).Create(deployment); err != nil {
					return err
				}
			} else {
				return errors.New("namespace is not defined for deployment when create")
			}
		} else if cm, ok := objs.(*v1.ConfigMap); ok {
			nm := cm.Namespace
			if len(nm) == 0 && len(namespaceName) != 0 {
				if _, err := clientset.CoreV1().ConfigMaps(namespaceName).Create(cm); err != nil {
					return err
				}
			} else if len(nm) != 0 {
				if _, err := clientset.CoreV1().ConfigMaps(nm).Create(cm); err != nil {
					return err
				}
			} else {
				return errors.New("namespace is not defined for configmaps when create")
			}
		} else if pod, ok := objs.(*v1.Pod); ok {
			nm := pod.Namespace
			if len(nm) == 0 && len(namespaceName) != 0 {
				if _, err := clientset.CoreV1().Pods(namespaceName).Create(pod); err != nil {
					return err
				}
			} else if len(nm) != 0 {
				if _, err := clientset.CoreV1().Pods(nm).Create(pod); err != nil {
					return err
				}
			} else {
				return errors.New("namespace is not defined for pods")
			}
		} else {
			return fmt.Errorf("unable to apply YAML. Unknown object type: %v", objs)
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

			klog.Warning("cleanUp: trying again...", i)
			time.Sleep(5 * time.Second)
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
			klog.Warning("Trying again in 5 seconds...", i)
			time.Sleep(5 * time.Second)
			continue
		}

		publicIP := ingress.Status.LoadBalancer.Ingress[0].IP
		if publicIP != "" {
			return publicIP, nil
		}

		klog.Warning("getPublicIP: trying again in 5 seconds...", i)
		time.Sleep(5 * time.Second)
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

func readBody(resp *http.Response) (string, error) {
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		return string(bodyBytes), nil
	}

	return "", nil
}

func getGateway() (*n.ApplicationGateway, error) {
	env := GetEnv()

	klog.Info("preparing app gateway client")
	client, err := getApplicationGatewaysClient()
	if err != nil {
		return nil, err
	}

	gateway, err := client.Get(
		context.TODO(),
		env.ResourceGroupName,
		env.AppGwName,
	)

	if err != nil {
		return nil, err
	}

	return &gateway, nil
}
