- [Assumptions](#assumptions)
- [Setting up Authentication with Azure Resource Manager (ARM)](#setting-up-authentication-with-azure-resource-manager)
    * [Setting up aad-pod-identity](#setting-up-aad-pod-identity)
        + [Create Azure Identity on ARM](#create-azure-identity-on-arm)
- [Install Ingress Controller using Helm](#install-ingress-controller-as-a-helm-chart)

# Setting up Application Gateway ingress controller on AKS
The Application Gateway Ingress controller runs as pod within the AKS cluster. It listens to [Kubernetes Ingress Resources](https://kubernetes.io/docs/concepts/services-networking/ingress/) from the Kubernetes API server and converts them to Azure Application Gateway configuration and updates the Application Gateway through the Azure Resource Manager (ARM).

In order to install the ingress controller on AKS we use [Helm](https://docs.microsoft.com/en-us/azure/aks/kubernetes-helm). 

## Assumptions
* An existing [Azure Application Gateway v2](https://docs.microsoft.com/en-us/azure/application-gateway/create-zone-redundant).
* An existing Azure Kubernetes Service cluster with the following properties:
    * RBAC is disabled in the cluster.
    * The AKS service is launched with Advanced Networking.
* The [aad-pod-identity](https://github.com/Azure/aad-pod-identity) service is installed on the AKS cluster.

## Setting up Authentication with Azure Resource Manager
Since the ingress controller needs to talk to the Kubernetes API server and the Azure Resource Manager it will need an identity to access both these entities. Since we are currently supporting only a non-RBAC cluster, the ingress controller currently does not need an identity to talk to the Kubernetes API server but needs and identity to talk to ARM. 

### Setting up aad-pod-identity
The [aad-pod-identity](https://github.com/Azure/aad-pod-identity) gives a clean way of exposing an existing Azure AD identity to a pod. Kindly follow the [aad-pod-identity installation instructions](https://github.com/Azure/aad-pod-identity#deploy-the-azure-aad-identity-infra) to deploy the aad-pod-identity service on your AKS cluster. This is a pre-requisite for installing the ingress controller.

#### Create Azure Identity on ARM

1. Create an Azure identity **in the same resource group as the AKS nodes** (typically the resource group with a `MC_` prefix string)

    ```bash
    az identity create -g <resourcegroup> -n <identity-name>
    ```
2. Find the principal, resource and client ID for this identity
    ```bash
    az identity show -g <resourcegroup> -n <identity-name>
    ```
3. Assign this new identity `Contributor` access on the application gateway
    ```bash
    az role assignment create --role Contributor --assignee <principal ID from the command above> --scope <Resource ID of Application Gateway>
    ```
4. Assign this new identity `Reader` access on the resource group that the application gateway belongs to
    ```bash
    az role assignment create --role Reader --assignee <principal ID from the command above> --scope <Resource ID of Application Gateway Resource Group>
    ```

## Install Ingress Controller as a Helm Chart

1. Add the `application-gateway-kubernetes-ingress` helm repo and perform a helm update

    ```bash
    helm repo add application-gateway-kubernetes-ingress https://azure.github.io/application-gateway-kubernetes-ingress/helm/
    helm repo update
    ```

2. Edit [helm-config.yaml](example/helm-config.yaml) and fill in the values for `appgw` and `armAuth`

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
    ```
    **NOTE:** The `<identity-resource-id>` and `<identity-client-id>` are the properties of the Azure AD Identity you setup in the previous section. You can retrieve this information by running the following command:
        ```bash
        az identity show -g <resourcegroup> -n <identity-name>
        ```
        Where `<resourcegroup>` is the resource group in which AKS cluster is running (this would have the prefix `MC_`).

3. Install the helm chart `application-gateway-kubernetes-ingress` with the `helm-config.yaml` configuration from the previous step

    ```bash
    helm install -f <helm-config.yaml> application-gateway-kubernetes-ingress/ingress-azure
    ```

4. Check the log of the newly created pod to verify if it started properly
