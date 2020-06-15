# Brownfield Deployment

The App Gateway Ingress Controller (AGIC) is a pod within your Kubernetes cluster.
AGIC monitors the Kubernetes [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/)
resources, and creates and applies App Gateway config based on these.

### Outline:
- [Prerequisites](#prerequisites)
- [Azure Resource Manager Authentication (ARM)](#azure-resource-manager-authentication)
    - Option 1: [Set up aad-pod-identity](#set-up-aad-pod-identity) and [Create Azure Identity on ARM](#create-azure-identity-on-arm)
    - Option 2: [Using a Service Principal](#using-a-service-principal)
- [Install Ingress Controller using Helm](#install-ingress-controller-as-a-helm-chart)
- [Multi-cluster / Shared App Gateway](#multi-cluster--shared-app-gateway): Install AGIC in an environment, where App Gateway is
shared between one or more AKS clusters and/or other Azure components.

### Prerequisites
This documents assumes you already have the following tools and infrastructure installed:
- [AKS](https://azure.microsoft.com/en-us/services/kubernetes-service/) with [Advanced Networking](https://docs.microsoft.com/en-us/azure/aks/configure-azure-cni) enabled
- [App Gateway v2](https://docs.microsoft.com/en-us/azure/application-gateway/create-zone-redundant) in the same virtual network as AKS
- [AAD Pod Identity](https://github.com/Azure/aad-pod-identity) installed on your AKS cluster
- [Cloud Shell](https://shell.azure.com/) is the Azure shell environment, which has `az` CLI, `kubectl`, and `helm` installed. These tools are required for the commands below.

Please __backup your App Gateway's configuration__ before installing AGIC:
  1. using [Azure Portal](https://portal.azure.com/) navigate to your `App Gateway` instance
  2. from `Export template` click `Download`

The zip file you downloaded will have JSON templates, bash, and PowerShell scripts you could use to restore App
Gateway should that become necessary

### Install Helm
[Helm](https://docs.microsoft.com/en-us/azure/aks/kubernetes-helm) is a package manager for
Kubernetes. We will leverage it to install the `application-gateway-kubernetes-ingress` package.
Use [Cloud Shell](https://shell.azure.com/) to install Helm:

1. Add the AGIC Helm repository:
    ```bash
    helm repo add application-gateway-kubernetes-ingress https://appgwingress.blob.core.windows.net/ingress-azure-helm-package/
    helm repo update
    ```

### Azure Resource Manager Authentication

AGIC communicates with the Kubernetes API server and the Azure Resource Manager. It requires an identity to access
these APIs.

### Set up AAD Pod Identity

[AAD Pod Identity](https://github.com/Azure/aad-pod-identity) is a controller, similar to AGIC, which also runs on your
AKS. It binds Azure Active Directory identities to your Kubernetes pods. Identity is required for an application in a
Kubernetes pod to be able to communicate with other Azure components. In the particular case here we need authorization
for the AGIC pod to make HTTP requests to [ARM](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-overview).

To install AAD Pod Identity to your cluster:

   - *RBAC enabled* AKS cluster

  ```bash
  kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/v1.5.5/deploy/infra/deployment-rbac.yaml
  ```

   - *RBAC disabled* AKS cluster

  ```bash
  kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/v1.5.5/deploy/infra/deployment.yaml
  ```

Next we need to create an Azure identity and give it permissions ARM.
Use [Cloud Shell](https://shell.azure.com/) to run all of the following commands and create an identity:

1. Create an Azure identity **in the same resource group as the AKS nodes**. Picking the correct resource group is
important. The resource group required in the command below is *not* the one referenced on the AKS portal pane. This is
the resource group of the `aks-agentpool` virtual machines. Typically that resource group starts with `MC_` and contains
 the name of your AKS. For instance: `MC_resourceGroup_aksABCD_westus`

    ```bash
    az identity create -g <agent-pool-resource-group> -n <identity-name>
    ```

1. For the role assignment commands below we need to obtain `clientId` for the newly created identity:

    ```bash
    az identity show -g <resourcegroup> -n <identity-name>
    ```

1. Give the identity `Contributor` access to you App Gateway. For this you need the ID of the App Gateway, which will
look something like this: `/subscriptions/A/resourceGroups/B/providers/Microsoft.Network/applicationGateways/C`

    Get the list of App Gateway IDs in your subscription with: `az network application-gateway list --query '[].id'`

    ```bash
    az role assignment create \
        --role Contributor \
        --assignee <clientId> \
        --scope <App-Gateway-ID>
    ```

1. Give the identity `Reader` access to the App Gateway resource group. The resource group ID would look like:
`/subscriptions/A/resourceGroups/B`. You can get all resource groups with: `az group list --query '[].id'`

    ```bash
    az role assignment create \
        --role Reader \
        --assignee <clientId> \
        --scope <App-Gateway-Resource-Group-ID>
    ```

### Using a Service Principal
It is also possible to provide AGIC access to ARM via a Kubernetes secret.

  1. Create an Active Directory Service Principal and encode with base64. The base64 encoding is required for the JSON
  blob to be saved to Kubernetes.

  ```bash
  az ad sp create-for-rbac --sdk-auth | base64 -w0
  ```

  2. Add the base64 encoded JSON blob to the `helm-config.yaml` file. More information on `helm-config.yaml` is in the
  next section.
   ```yaml
   armAuth:
       type: servicePrincipal
       secretJSON: <Base64-Encoded-Credentials>
   ```

## Install Ingress Controller as a Helm Chart
You can use [Cloud Shell](https://shell.azure.com/) to install the AGIC Helm package:

1. Add the `application-gateway-kubernetes-ingress` helm repo and perform a helm update

    ```bash
    helm repo add application-gateway-kubernetes-ingress https://appgwingress.blob.core.windows.net/ingress-azure-helm-package/
    helm repo update
    ```

1. Download [helm-config.yaml](../examples/sample-helm-config.yaml), which will configure AGIC:
    ```bash
    wget https://raw.githubusercontent.com/Azure/application-gateway-kubernetes-ingress/master/docs/examples/sample-helm-config.yaml -O helm-config.yaml
    ```

1. Edit [helm-config.yaml](../examples/sample-helm-config.yaml) and fill in the values for `appgw` and `armAuth`.
    ```bash
    nano helm-config.yaml
    ```

    **NOTE:** The `<identity-resource-id>` and `<identity-client-id>` are the properties of the Azure AD Identity you setup in the previous section. You can retrieve this information by running the following command: `az identity show -g <resourcegroup> -n <identity-name>`, where `<resourcegroup>` is the resource group in which the top level AKS cluster object, Application Gateway and Managed Identify are deployed.

1. Install Helm chart `application-gateway-kubernetes-ingress` with the `helm-config.yaml` configuration from the previous step

    ```bash
    helm install ingress-azure \
      -f helm-config.yaml \
      application-gateway-kubernetes-ingress/ingress-azure \
      --version 1.0.0
    ```

    >Note: Use at least version 1.2.0-rc3, e.g. `--version 1.2.0-rc3`, when installing on k8s version >= 1.16

    Alternatively you can combine the `helm-config.yaml` and the Helm command in one step:
    ```bash
    helm install ingress-azure application-gateway-kubernetes-ingress/ingress-azure \
         --namespace default \
         --debug \
         --set appgw.name=applicationgatewayABCD \
         --set appgw.resourceGroup=your-resource-group \
         --set appgw.subscriptionId=subscription-uuid \
         --set appgw.usePrivateIP=false \
         --set appgw.shared=false \
         --set armAuth.type=servicePrincipal \
         --set armAuth.secretJSON=$(az ad sp create-for-rbac --sdk-auth | base64 -w0) \
         --set rbac.enabled=true \
         --set verbosityLevel=3 \
         --set kubernetes.watchNamespace=default \
         --version 1.0.0
    ```

    >Note: Use at least version 1.2.0-rc3, e.g. `--version 1.2.0-rc3`, when installing on k8s version >= 1.16

1. Check the log of the newly created pod to verify if it started properly

Refer to the [tutorials](../tutorial.md) to understand how you can expose an AKS service over HTTP or HTTPS, to the internet, using an Azure App Gateway.



## Multi-cluster / Shared App Gateway
By default AGIC assumes full ownership of the App Gateway it is linked to. AGIC version 0.8.0 and later can
share a single App Gateway with other Azure components. For instance, we could use the same App Gateway for an app
hosted on VMSS as well as an AKS cluster.

Please __backup your App Gateway's configuration__ before enabling this setting:
  1. using [Azure Portal](https://portal.azure.com/) navigate to your `App Gateway` instance
  2. from `Export template` click `Download`

The zip file you downloaded will have JSON templates, bash, and PowerShell scripts you could use to restore App Gateway

### Example Scenario
Let's look at an imaginary App Gateway, which manages traffic for 2 web sites:
  - `dev.contoso.com` - hosted on a new AKS, using App Gateway and AGIC
  - `prod.contoso.com` - hosted on an [Azure VMSS](https://azure.microsoft.com/en-us/services/virtual-machine-scale-sets/)

With default settings, AGIC assumes 100% ownership of the App Gateway it is pointed to. AGIC overwrites all of App
Gateway's configuration. If we were to manually create a listener for `prod.contoso.com` (on App Gateway), without
defining it in the Kubernetes Ingress, AGIC will delete the `prod.contoso.com` config within seconds.

To install AGIC and also serve `prod.contoso.com` from our VMSS machines, we must constrain AGIC to configuring
`dev.contoso.com` only. This is facilitated by instantiating the following
[CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/):

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

The command above creates an `AzureIngressProhibitedTarget` object. This makes AGIC (version 0.8.0 and later) aware of the existence of
App Gateway config for `prod.contoso.com` and explicitly instructs it to avoid changing any configuration
related to that hostname.


### Enable with new AGIC installation
To limit AGIC (version 0.8.0 and later) to a subset of the App Gateway configuration modify the `helm-config.yaml` template.
Under the `appgw:` section, add `shared` key and set it to to `true`.

```yaml
appgw:
    subscriptionId: <subscriptionId>    # existing field
    resourceGroup: <resourceGroupName>  # existing field
    name: <applicationGatewayName>      # existing field
    shared: true                        # <<<<< Add this field to enable shared App Gateway >>>>>
```

Apply the Helm changes:
  1. Ensure the `AzureIngressProhibitedTarget` CRD is installed with:
      ```bash
      kubectl apply -f https://raw.githubusercontent.com/Azure/application-gateway-kubernetes-ingress/ae695ef9bd05c8b708cedf6ff545595d0b7022dc/crds/AzureIngressProhibitedTarget.yaml
      ```
  2. Update Helm:
      ```bash
      helm upgrade \
          --recreate-pods \
          -f helm-config.yaml \
          ingress-azure application-gateway-kubernetes-ingress/ingress-azure
      ```

As a result your AKS will have a new instance of `AzureIngressProhibitedTarget` called `prohibit-all-targets`:
```bash
kubectl get AzureIngressProhibitedTargets prohibit-all-targets -o yaml
```

The object `prohibit-all-targets`, as the name implies, prohibits AGIC from changing config for *any* host and path.
Helm install with `appgw.shared=true` will deploy AGIC, but will not make any changes to App Gateway.


### Broaden permissions
Since Helm with `appgw.shared=true` and the default `prohibit-all-targets` blocks AGIC from applying any config.

Broaden AGIC permissions with:
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

2. Only after you have created your own custom prohibition, you can delete the default one, which is too broad:

    ```bash
    kubectl delete AzureIngressProhibitedTarget prohibit-all-targets
    ```

### Enable for an existing AGIC installation
Let's assume that we already have a working AKS, App Gateway, and configured AGIC in our cluster. We have an Ingress for
`prod.contosor.com` and are successfully serving traffic for it from AKS. We want to add `staging.contoso.com` to our
existing App Gateway, but need to host it on a [VM](https://azure.microsoft.com/en-us/services/virtual-machines/). We
are going to re-use the existing App Gateway and manually configure a listener and backend pools for
`staging.contoso.com`. But manually tweaking App Gateway config (via
[portal](https://portal.azure.com), [ARM APIs](https://docs.microsoft.com/en-us/rest/api/resources/) or
[Terraform](https://www.terraform.io/)) would conflict with AGIC's assumptions of full ownership. Shortly after we apply
changes, AGIC will overwrite or delete them.

We can prohibit AGIC from making changes to a subset of configuration.

1. Create an `AzureIngressProhibitedTarget` object:
    ```bash
    cat <<EOF | kubectl apply -f -
    apiVersion: "appgw.ingress.k8s.io/v1"
    kind: AzureIngressProhibitedTarget
    metadata:
      name: manually-configured-staging-environment
    spec:
      hostname: staging.contoso.com
    EOF
    ```

2. View the newly created object:
    ```bash
    kubectl get AzureIngressProhibitedTargets
    ```

3. Modify App Gateway config via portal - add listeners, routing rules, backends etc. The new object we created
(`manually-configured-staging-environment`) will prohibit AGIC from overwriting App Gateway configuration related to
`staging.contoso.com`.
