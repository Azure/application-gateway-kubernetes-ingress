# Table of Contents
- [Deploying the infrastructure on Azure](#deploying-the-infrastructure-on-azure)
- [Setting up Application Gateway Ingress Controller on AKS](#setting-up-application-gateway-ingress-controller-on-aks)

## Deploying the infrastructure on Azure

To create the pre-requisite Azure resources, you can use the following template. It creates:
1) Azure Virtual Network with 2 subnets.
2) Azure Application Gateway v2.
3) Azure Kubernetes Service cluster with required permission to deploy nodes in the Virtual Network. You have an option to deploy RBAC enabled AKS cluster
4) User Assigned Identity to initialize the aad-pod-identity service and ingress controller.
5) Set required RBACs.


## Prerequisites

The steps below require the following software to be installed on your workstation:
  1. `az` - Azure CLI: [installation instructions](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest)
  1. `kubectl` - Kubernetes command-line tool: [installation instructions](https://kubernetes.io/docs/tasks/tools/install-kubectl)
  1. `helm` - tool for managing pre-configured Kubernetes resources: [installation instructions](https://github.com/helm/helm/releases/latest)


## Steps

1) Create a service principal that will be assigned to the AKS cluster in the template.
    ```bash
    az ad sp create-for-rbac --skip-assignment
    az ad sp show --id <appId> --query "objectId"
    ```
    **Note the appId, password and objectId.**

2) After creating the service principal in the step above, click to create a custom template deployment. Provide the appId for servicePrincipalClientId, password and objectId in the parameters.
    Note: For deploying an *RBAC* enabled cluster, set `aksEnabledRBAC` parameter to `true`.

    <a href="https://portal.azure.com/#create/Microsoft.Template/uri/https%3A%2F%2Fraw.githubusercontent.com%2FAzure%2Fapplication-gateway-kubernetes-ingress%2Fmaster%2Fdeploy%2Fazuredeploy.json" target="_blank">
        <img src="http://azuredeploy.net/deploybutton.png"/>
    </a>
    <a href="http://armviz.io/#/?load=https%3A%2F%2Fraw.githubusercontent.com%2FAzure%2Fapplication-gateway-kubernetes-ingress%2Fmaster%2Fdeploy%2Fazuredeploy.json" target="_blank">
        <img src="http://armviz.io/visualizebutton.png"/>
    </a>

    After the deployment completes, you can look at the Output window for parameters needed in the following steps.

    Example:
    ![Deployment Output](images/deployment-output.png)

## Setting up Application Gateway Ingress Controller on AKS

### Overview
With the instructions in the previous section we created and configured a new Azure Kubernetes Service (AKS) cluster. We are now ready to deploy new Kubernetes infrastructure. The instructions below will guide us through the proccess of installing the following 2 components on our new AKS:
1. **Azure Active Directory Pod Identity** - Provides token-based access to the [Azure Resource Manager (ARM)](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-overview) via user-assigned identity. Adding this system will result in the installation of the following within your AKS cluster:
   1. custom Kubernetes resource definitions: `AzureIdentity`, `AzureAssignedIdentity`, `AzureIdentityBinding`
   1. [Managed Identity Controller (MIC)](https://github.com/Azure/aad-pod-identity#managed-identity-controllermic) component
   1. [Node Managed Identity (NMI)](https://github.com/Azure/aad-pod-identity#node-managed-identitynmi) component 
1. **Application Gateway Ingress Controller** - This is the controller which monitors ingress-related events and actively keeps your Azure Application Gateway installation in sync with the changes within the AKS cluster.

Steps:

1) To configure kubectl to connect to the deployed Azure Kubernetes Cluster, follow these [instructions](https://docs.microsoft.com/en-us/azure/aks/kubernetes-walkthrough#connect-to-the-cluster).

2) Add aad pod identity service to the cluster using the following command. This service will be used by the ingress controller. You can refer [aad-pod-identity](https://github.com/Azure/aad-pod-identity) for more information.  
    * *RBAC disabled* AKS cluster -
    ```bash
    kubectl create -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment.yaml
    ```

    * *RBAC enabled* AKS cluster -
    ```bash
    kubectl create -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml
    ```

3) Install [Helm](https://docs.microsoft.com/en-us/azure/aks/kubernetes-helm) and run the following to add `application-gateway-kubernetes-ingress` helm package:

    * *RBAC disabled* AKS cluster -
    ```bash
    helm init
    helm repo add application-gateway-kubernetes-ingress https://azure.github.io/application-gateway-kubernetes-ingress/helm/
    helm repo update
    ```

    * *RBAC enabled* AKS cluster -
    ```bash
    kubectl create serviceaccount --namespace kube-system tiller-sa
    kubectl create clusterrolebinding tiller-cluster-rule --clusterrole=cluster-admin --serviceaccount=kube-system:tiller-sa
    helm init --tiller-namespace kube-system --service-account tiller-sa
    helm repo add application-gateway-kubernetes-ingress https://azure.github.io/application-gateway-kubernetes-ingress/helm/
    helm repo update
    ```

4) Edit [helm-config.yaml](example/helm-config.yaml) and fill in the values for `appgw` and `armAuth`
    ```yaml
    # This file contains the essential configs for the ingress controller helm chart

    ################################################################################
    # Specify which application gateway the ingress controller will manage
    #
    appgw:
        subscriptionId: <subscription-id>
        resourceGroup: <resourcegroup-name>
        name: <applicationgateway-name>

    ################################################################################
    # Specify which kubernetes namespace the ingress controller will watch
    # Default value is "default"
    #
    # kubernetes:
    #   watchNamespace: <namespace>

    ################################################################################
    # Specify the authentication with Azure Resource Manager
    #
    # Two authentication methods are available:
    # - Option 1: AAD-Pod-Identity (https://github.com/Azure/aad-pod-identity)
    armAuth:
        type: aadPodIdentity
        identityResourceID: <identity-resource-id>
        identityClientID:  <identity-client-id>

    ################################################################################
    # Specify if the cluster is RBAC enabled or not
    rbac:
        enabled: false # true/false

    ################################################################################
    # Specify aks cluster related information
    aksClusterConfiguration:
        apiServerAddress: <aks-api-server-address>
    ```
    **NOTE:** The `<identity-resource-id>` and `<identity-client-id>` are the properties of the Azure AD Identity you setup in the previous section. You can retrieve this information by running the following command:  
        ```bash
        az identity show -g <resourcegroup> -n <identity-name>
        ```  
        Where `<resourcegroup>` is the resource group in which the top level AKS cluster object, Application Gateway and Managed Identify are deployed.

    Then execute the following to the install the Application Gateway ingress controller package.  
    ```bash
    helm install -f helm-config.yaml application-gateway-kubernetes-ingress/ingress-azure
    ```

Refer to the [tutorials](tutorial.md) to understand how you can expose an AKS service over HTTP or HTTPS, to the internet, using an Azure Application Gateway.
