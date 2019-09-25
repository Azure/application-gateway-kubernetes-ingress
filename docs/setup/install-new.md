# Greenfield Deployment

The instructions below assume Application Gateway Ingress Controller (AGIC) will be
installed in an environment with no pre-existing components.

### Required Command Line Tools

We recommend the use of [Azure Cloud Shell](https://shell.azure.com/) for all command line operations below. Launch your shell from shell.azure.com or by clicking the link:

[![Embed launch](https://shell.azure.com/images/launchcloudshell.png "Launch Azure Cloud Shell")](https://shell.azure.com)

Alternatively, launch Cloud Shell from Azure portal using the following icon:

![Portal launch](../portal-launch-icon.png)

Your [Azure Cloud Shell](https://shell.azure.com/) already has all necessary tools. Should you
choose to use another environment, please ensure the following command line tools are installed:

1. `az` - Azure CLI: [installation instructions](https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest)
1. `kubectl` - Kubernetes command-line tool: [installation instructions](https://kubernetes.io/docs/tasks/tools/install-kubectl)
1. `helm` - Kubernetes package manager: [installation instructions](https://github.com/helm/helm/releases/latest)


### Create an Identity

Follow the steps below to create an Azure Active Directory (AAD) [service principal object](https://docs.microsoft.com/en-us/azure/active-directory/develop/app-objects-and-service-principals#service-principal-object). Please record the `appId`, `password`, and `objectId` values - these will be used in the following steps.

1. Create AD service principal ([Read more about RBAC](https://docs.microsoft.com/en-us/azure/role-based-access-control/overview)):
    ```bash
    az ad sp create-for-rbac --skip-assignment
    ```
    note: the `appId` and `password` values from the JSON output will be used in the following steps


2. Use the `appId` from the previous command's output to get the `objectId` of the newl service principal:
    ```bash
    az ad sp show --id <appId> --query "objectId"
    ```
    note: the output of this command is `objectId`, which will be used in the ARM template below

### Deploy Components
Click on the **Deploy to Azure** icon below to begin the infrastructure deployment using an [ARM template](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-authoring-templates). This step will add the following components to your subscription:

- [Azure Kubernetes Service](https://docs.microsoft.com/en-us/azure/aks/intro-kubernetes)
- [Application Gateway](https://docs.microsoft.com/en-us/azure/application-gateway/overview) v2
- [Virtual Network](https://docs.microsoft.com/en-us/azure/virtual-network/virtual-networks-overview) with 2 [subnets](https://docs.microsoft.com/en-us/azure/virtual-network/virtual-networks-overview)
- [Public IP Address](https://docs.microsoft.com/en-us/azure/virtual-network/virtual-network-public-ip-address)
- [Managed Identity](https://docs.microsoft.com/en-us/azure/active-directory/managed-identities-azure-resources/overview), which will be used by [AAD Pod Identity](https://github.com/Azure/aad-pod-identity/blob/master/README.md)

#### Important
Please use the `appId`, `objectId`, and `password` values from the `az` commands above and
paste them in the corresponding ARM template fields:
  - paste the `appId` vaule in the `Aks Service Principal App Id` template field
  - paste the `password` value in the `Aks Service Principal Client Secret` field
  - paste the `objectId` value in the `Aks Service Principal Object Id` field

Note: To deploy an **RBAC** enabled cluster, set the `aksEnabledRBAC` field to `true`

<a href="https://portal.azure.com/#create/Microsoft.Template/uri/https%3A%2F%2Fraw.githubusercontent.com%2FAzure%2Fapplication-gateway-kubernetes-ingress%2Fmaster%2Fdeploy%2Fazuredeploy.json" target="_blank">
<img src="http://azuredeploy.net/deploybutton.png"/>
</a>

Navigate to the deployment output and record the parameters:
[Azure portal](https://portal.azure.com/): `Home -> *resource group* -> Deployments -> *new deployment* -> Outputs`)

Example: ![Deployment Output](../images/deployment-output.png)

## Set up Application Gateway Ingress Controller

### Overview

With the instructions in the previous section we created and configured a new Azure Kubernetes Service (AKS) cluster. We are now ready to deploy to our new Kubernetes infrastructure. The instructions below will guide us through the proccess of installing the following 2 components on our new AKS:

1. **Azure Active Directory Pod Identity** - Provides token-based access to the [Azure Resource Manager (ARM)](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-overview) via user-assigned identity. Adding this system will result in the installation of the following within your AKS cluster:
   1. Custom Kubernetes resource definitions: `AzureIdentity`, `AzureAssignedIdentity`, `AzureIdentityBinding`
   1. [Managed Identity Controller (MIC)](https://github.com/Azure/aad-pod-identity#managed-identity-controllermic) component
   1. [Node Managed Identity (NMI)](https://github.com/Azure/aad-pod-identity#node-managed-identitynmi) component
1. **Application Gateway Ingress Controller** - This is the controller which monitors ingress-related events and actively keeps your Azure Application Gateway installation in sync with the changes within the AKS cluster.

Steps:

1. Configure `kubectl` with access to your newly deployed AKS:
```bash
az aks get-credentials --resource-group <your-new-resource-group> --name <name-of-new-AKS-cluster>
```
[More on setting up kubectl](https://docs.microsoft.com/en-us/azure/aks/kubernetes-walkthrough#connect-to-the-cluster).

1. Add aad pod identity service to the cluster using the following command. This service will be used by the ingress controller. You can refer [aad-pod-identity](https://github.com/Azure/aad-pod-identity) for more information.

    - *RBAC disabled* AKS cluster

    ```bash
    kubectl create -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment.yaml
    ```

    - *RBAC enabled* AKS cluster

    ```bash
    kubectl create -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml
    ```

1. Install [Helm](https://docs.microsoft.com/en-us/azure/aks/kubernetes-helm) and run the following to add `application-gateway-kubernetes-ingress` helm package:

    - *RBAC disabled* AKS cluster

    ```bash
    helm init
    helm repo add application-gateway-kubernetes-ingress https://appgwingress.blob.core.windows.net/ingress-azure-helm-package/
    helm repo update
    ```

    - *RBAC enabled* AKS cluster

    ```bash
    kubectl create serviceaccount --namespace kube-system tiller-sa
    kubectl create clusterrolebinding tiller-cluster-rule --clusterrole=cluster-admin --serviceaccount=kube-system:tiller-sa
    helm init --tiller-namespace kube-system --service-account tiller-sa
    helm repo add application-gateway-kubernetes-ingress https://appgwingress.blob.core.windows.net/ingress-azure-helm-package/
    helm repo update
    ```

1. Edit [helm-config.yaml](../examples/sample-helm-config.yaml) and fill in the values for `appgw` and `armAuth`

    ```yaml
    # This file contains the essential configs for the ingress controller helm chart

    # Verbosity level of the App Gateway Ingress Controller
    verbosityLevel: 3

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
    # Leaving this variable out or setting it to blank or empty string would
    # result in Ingress Controller observing all acessible namespaces.
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
    # Specify aks cluster related information. THIS IS BEING DEPRECATED.
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

Jump next to **[tutorials](../tutorial.md)** to understand how you can expose an AKS service over HTTP or HTTPS, to the internet, using an Azure Application Gateway.
