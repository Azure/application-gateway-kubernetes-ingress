# Brownfield Deployment

The App Gateway Ingress (AGIC) controller runs as a separate pod within your Kubernetes cluster. AGIC monitors the
[Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) resources and composes App Gateway
configuration based on these. AGIC applies App Gateway config via the Azure Resource Manager (ARM).

### Outline:
- [Prerequisites](#prerequisites)
- [Azure Resource Manager Authentication (ARM)](#azure-resource-manager-authentication)
    - Option 1: [Set up aad-pod-identity](#set-up-aad-pod-identity) and [Create Azure Identity on ARM](#create-azure-identity-on-arm)
    - Option 2: [Using a Service Principal](#using-a-service-principal)
- [Install Ingress Controller using Helm](#install-ingress-controller-as-a-helm-chart)


### Prerequisites
This documents assumes you already have the following tools and infrastructure installed:
- [Helm](https://docs.microsoft.com/en-us/azure/aks/kubernetes-helm) installed on your local machine
- [AKS](https://azure.microsoft.com/en-us/services/kubernetes-service/) with [Advanced Networking](https://docs.microsoft.com/en-us/azure/aks/configure-azure-cni) enabled
- [App Gateway v2](https://docs.microsoft.com/en-us/azure/application-gateway/create-zone-redundant) in the same virtual network as AKS
- [AAD Pod Identity](https://github.com/Azure/aad-pod-identity) installed on your AKS cluster

### Azure Resource Manager Authentication

AGIC communicates with the Kubernetes API server and the Azure Resource Manager. It requires an identity to access
these APIs.

### Set up AAD Pod Identity

[AAD Pod Identity](https://github.com/Azure/aad-pod-identity) is a controller, similar to AGIC, which also runs on your
AKS. It binds Azure Active Directory identities to your Kubernetes pods. Identity is required for an application in a
Kubernetes pod to be able to communicate with other Azure components. In the particular case here we need authorization
for the AGIC pod to make HTTP requests to [ARM](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-overview).

Follow the [AAD Pod Identity installation instructions](https://github.com/Azure/aad-pod-identity#deploy-the-azure-aad-identity-infra) to add this component to your AKS.

Next we need to create an Azure identity and give it permissions ARM

1. Create an Azure identity **in the same resource group as the AKS nodes**. Picking the correct resource group is
important. The resource group required in the command below is *not* the one referenced on the AKS portal pane. This is
the resource group of the `aks-agentpool` virtual machines. Typically that resource group starts with `MC_` and contains
 the name of your AKS. For instance: `MC_resourceGroup_aksABCD_westus`

    ```bash
    az identity create -g <agent-pool-resource-group> -n <identity-name>
    ```

1. For the role assignment commands below we need to obtain `principalId` for the newly created identity:

    ```bash
    az identity show -g <resourcegroup> -n <identity-name>
    ```

1. Give the identity `Contributor` access to you App Gateway. For this you need the ID of the App Gateway, which will
look something like this: `/subscriptions/A/resourceGroups/B/providers/Microsoft.Network/applicationGateways/C`

    Get the list of App Gateway IDs in your subscription with: `az network application-gateway list --query '[].id'` 

    ```bash
    az role assignment create \
        --role Contributor \
        --assignee <principalId> \
        --scope <App-Gateway-ID>
    ```

1. Give the identity `Reader` access to the App Gateway resource group. The resource group ID would look like:
`/subscriptions/A/resourceGroups/B`. You can get all resource groups with: `az group list --query '[].id'` 

    ```bash
    az role assignment create \
        --role Reader \
        --assignee <principalId> \
        --scope <App-Gateway-Resource-Group-ID>
    ```

### Using a Service Principal
It is also possible to provide AGIC access to ARM via a Kubernetes secret. 

  1. Create an Active Directory Service Principal and encode with base64. The base64 encoding is required for the JSON
  blob to be saved to Kubernetes.

  ```bash
  az ad sp create-for-rbac --subscription <subscription-uuid> --sdk-auth | base64 -w0
  ```

  2. Add the base64 encoded JSON blob to the `helm-config.yaml` file. More information on `helm-config.yaml` is in the
  next section.
   ```yaml
   armAuth:
       type: servicePrincipal
       secretJSON: <Base64-Encoded-Credentials>
   ```

## Install Ingress Controller as a Helm Chart

1. Add the `application-gateway-kubernetes-ingress` helm repo and perform a helm update

    ```bash
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

    # - Option 2: Service Principal Credentials
    # armAuth:
    #     type: servicePrincipal
    #     secretJSON: <<Generate this value with: "az ad sp create-for-rbac --subscription <subscription-uuid> --sdk-auth | base64 -w0">>

    ################################################################################
    # Specify if the cluster is RBAC enabled or not
    rbac:
        enabled: false # true/false

    ################################################################################
    # Specify aks cluster related information. THIS IS BEING DEPRECATED.
    aksClusterConfiguration:
        apiServerAddress: <aks-api-server-address>

    # Setting brownfield.enabled to "true" will create an AzureIngressProhibitedTarget CRD.
    # This blacklist all hostnames and paths, prohibiting AGIC from applying config for any of them.
    # Use "kubectl get AzureIngressProhibitedTargets" to view and change prohibited targets
    brownfield:
        enabled: false
    ```

    **NOTE:** The `<identity-resource-id>` and `<identity-client-id>` are the properties of the Azure AD Identity you
    setup in the previous section. Run the following command to obtain these values:

    ```bash
    az identity show -g <resourcegroup> -n <identity-name>
    ```

    Where `<resourcegroup>` is the resource group in which AKS cluster is running (this would have the prefix `MC_`).

1. Install Helm chart `application-gateway-kubernetes-ingress` with the `helm-config.yaml` configuration from the previous step

    ```bash
    helm install -f <helm-config.yaml> application-gateway-kubernetes-ingress/ingress-azure
    ```

    Alternatively you can combine the `helm-config.yaml` and the Helm command in one step:
    ```bash
    helm install ./helm/ingress-azure \
         --name ingress-azure \
         --version 0.7.1 \
         --namespace default \
         --debug \
         --set appgw.name=applicationgatewayABCD \
         --set appgw.resourceGroup=your-resource-group \
         --set appgw.subscriptionId=subscription-uuid \
         --set armAuth.type=servicePrincipal \
         --set armAuth.secretJSON=$(az ad sp create-for-rbac --subscription <subscription-uuid> --sdk-auth | base64 -w0) \
         --set rbac.enabled=true \
         --set verbosityLevel=3 \
         --set kubernetes.watchNamespace=default \
         --set aksClusterConfiguration.apiServerAddress=aks-abcdefg.hcp.westus2.azmk8s.io \
         --set brownfield.enabled=false
    ```

1. Check the log of the newly created pod to verify if it started properly

Refer to the [tutorials](../tutorial.md) to understand how you can expose an AKS service over HTTP or HTTPS, to the internet, using an Azure App Gateway.