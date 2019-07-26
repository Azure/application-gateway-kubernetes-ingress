# Brownfield Deployment

## Table of Contents

- [Prerequisites](#prerequisites)
- [Set up Authentication with Azure Resource Manager (ARM)](#set-up-authentication-with-azure-resource-manager)
    - Option 1: [Setting up aad-pod-identity](#setting-up-aad-pod-identity)
        -[Create Azure Identity on ARM](#create-azure-identity-on-arm)
    - Option 2: [Using a Service Principal](#using-a-service-principal)
- [Install Ingress Controller using Helm](#install-ingress-controller-as-a-helm-chart)
- [Hybrid Environment](hybrid-environment)

## Set up Application Gateway ingress controller on AKS

The Application Gateway Ingress controller runs as pod in your AKS. AGIC listens to
[Kubernetes Ingress Resources](https://kubernetes.io/docs/concepts/services-networking/ingress/) and transforms these
to Application Gateway configuration. App Gateway config is applied via the Azure Resource Manager (ARM).

### Prerequisites

- [Helm](https://docs.microsoft.com/en-us/azure/aks/kubernetes-helm) installed on your local machine
- existing [Azure Application Gateway v2](https://docs.microsoft.com/en-us/azure/application-gateway/create-zone-redundant)
- existing [AKS](https://azure.microsoft.com/en-us/services/kubernetes-service/) with [Advanced Networking](https://docs.microsoft.com/en-us/azure/aks/configure-azure-cni) enabled
- The [aad-pod-identity](https://github.com/Azure/aad-pod-identity) service is installed on the AKS cluster.

### Set up Authentication with Azure Resource Manager

AGIC communicates with the Kubernetes API server and the Azure Resource Manager. It requires an identity to access
these APIs.

### Setting up aad-pod-identity

The [aad-pod-identity](https://github.com/Azure/aad-pod-identity) gives a clean way of exposing an existing Azure AD identity to a pod. Kindly follow the [aad-pod-identity installation instructions](https://github.com/Azure/aad-pod-identity#deploy-the-azure-aad-identity-infra) to deploy the aad-pod-identity service on your AKS cluster. This is a pre-requisite for installing the ingress controller.

#### Create Azure Identity on ARM

1. Create an Azure identity **in the same resource group as the AKS nodes** (typically the resource group with a `MC_` prefix string)

    ```bash
    az identity create -g <resourcegroup> -n <identity-name>
    ```

1. Find the principal, resource and client ID for this identity

    ```bash
    az identity show -g <resourcegroup> -n <identity-name>
    ```

1. Assign this new identity `Contributor` access on the application gateway

    ```bash
    az role assignment create --role Contributor --assignee <principal ID from the command above> --scope <Resource ID of Application Gateway>
    ```

1. Assign this new identity `Reader` access on the resource group that the application gateway belongs to

    ```bash
    az role assignment create --role Reader --assignee <principal ID from the command above> --scope <Resource ID of Application Gateway Resource Group>
    ```
### Using a Service Principal
  Create Service Principal credentials with:

  ```bash
  az ad sp create-for-rbac --subscription <subscription-uuid> --sdk-auth | base64 -w0
  ```

  Add the base64 encoded JSON blob to the `helm-config.yaml` file (described in the next section):
   ```yaml
   armAuth:
     type: servicePrincipal
     secretJSON: <base-64-encoded-credentials>
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

Refer to the [tutorials](../tutorial.md) to understand how you can expose an AKS service over HTTP or HTTPS, to the internet, using an Azure Application Gateway.

## Hybrid Environment
We have the option to deploy AGIC in an existing AKS and constrain it so it controls a subset of the traffic from App Gateway to AKS.

### Scenario
Let's take an imaginary App Gatway, which manages traffic for 2 web sites:
  - prod.contoso.com
  - dev.contoso.com

A new assignment requires us to:
  - start serving `dev.contoso.com` from a new AKS, using App Gateway and the Ingress Controller
  - continue serving `prod.contoso.com` from an existing [Azure VMSS](https://azure.microsoft.com/en-us/services/virtual-machine-scale-sets/)

Until now (and by default) AGIC assumes 100% ownership of the configuration of the App Gateway, to which it connects. AGIC overwrites all of App Gateway's configuration. If `prod.contoso.com` is not defined in the Kubernetes Ingress, the config for `prod.contoso.com` (listeners, routing rules, backends etc.) will be deleted.

To install AGIC and continue serving `prod.contoso.com` from our VMSS machines - we must create the following [CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/):

```bash
cat <<EOF | kubectl apply -f -
apiVersion: "appgw.ingress.k8s.io/v1"
kind: AzureIngressProhibitedTarget
metadata:
  name: prod-contoso-com
spec:
  hostname: prod.contoso.com
EOF
```

The bash command above will create an `AzureIngressProhibitedTarget` object. This makes AGIC aware of the existence of
App Gateway config for `prod.contoso.com`. This explicitly instructs App Gateway to avoid changing any configuration related to `hostname` in the command.


### Enable it on your AGIC
To limit AGIC to a subset of the configuration of your App Gateway, we will tweak the `helm-config.yaml` template. Set `brownfield.enabled` to "true" in the `helm-config.yaml` we created already. With this addition Helm will:
  - create a new CRD on your AKS: `AzureIngressProhibitedTarget`
  - create new instance of `AzureIngressProhibitedTarget` called `prohibit-all-targets`

The default `prohibit-all-targets` setup prohibits AGIC from changing config for any host and any path. Helm install
with `brownfield.enabled=true` will deploy AGIC, but will not make any changes to App Gateway.


### Broaden permissions
After deployment you will have the default prohibition `prohibit-all-targets`:

```bash
kubectl get AzureIngressProhibitedTarget
```

View the contents of the object:
```bash
kubectl get AzureIngressProhibitedTarget prohibit-all-targets -o yaml
```

You can broaden the permissions AGIC has:
  1. Create a new `AzureIngressProhibitedTarget` with your specific setup:
```bash
cat <<EOF | kubectl apply -f -
apiVersion: "appgw.ingress.k8s.io/v1"
kind: AzureIngressProhibitedTarget
metadata:
  name: your-custom-prohibitions
spec:
  hostname: your.own-hostname.com
EOF
```
2. Only after you have created your specific prohibition, delete the default one:

```bash
kubectl delete AzureIngressProhibitedTarget prohibit-all-targets
```
