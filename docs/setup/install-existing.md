# Brownfield Deployment

The App Gateway Ingress Controller (AGIC) is a pod within your Kubernetes cluster.
AGIC monitors the Kubernetes [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/)
resources, and creates and applies App Gateway config based on these.

### Outline:
- [Prerequisites](#prerequisites)
- [Setup Azure Resource Manager Authentication (ARM)](#setup-azure-resource-manager-authentication-arm)
    - Option 1: [Using User assigned Identity](#using-user-assigned-identity)
    - Option 2: [Using a Service Principal](#using-service-principal)
- [Install Ingress Controller using Helm](#install-ingress-controller-helm-chart)
- [Multi-cluster / Shared App Gateway](../features/shared-app-gateway): Install AGIC in an environment, where App Gateway is
shared between one or more AKS clusters and/or other Azure components.
- [Install a Sample App](#install-a-sample-app)

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
1. `jq` - command-line JSON processor: [installation instructions](https://stedolan.github.io/jq/download/)

## Setup variables

```bash
# provide information about pre-existing AKS cluster and Application Gateway
applicationGatewayName="<application-gateway-name>"
applicationGatewayGroupName="<application-gateway-group-name>"
aksClusterName="<aks-cluster-name>"
aksClusterGroupName="<aks-cluster-group-name>"

# name of the user assigned identity which will be created below
agicIdentityName="agic-identity"
```

## Setup Azure Resource Manager Authentication (ARM)

### Using User assigned Identity

1. Create a User assigned Identity in AKS Agent Pool's resource group
    ```bash
    # Create identity in agent pool's resource group
    nodeResourceGroupName=$(az aks show -n $aksClusterName -g $aksClusterGroupName --query "nodeResourceGroup" -o tsv)

    az identity create -n $agicIdentityName -g $nodeResourceGroupName -l $location
    identityPrincipalId=$(az identity show -n $agicIdentityName -g $nodeResourceGroupName --query "principalId" -o tsv)
    identityResourceId=$(az identity show -n $agicIdentityName -g $nodeResourceGroupName --query "id" -o tsv)
    identityClientId=$(az identity show -n $agicIdentityName -g $nodeResourceGroupName --query "clientId" -o tsv)
    ```

1. Assign `Contributor` role to the Application Gateway resource group. If this step fails with "No matches in graph database for ...", then try again after few seconds.
    ```bash
    applicationGatewayGroupId=$(az group show -g $applicationGatewayGroupName -o tsv --query "id")

    az role assignment create \
            --role Contributor \
            --assignee "$identityPrincipalId" \
            --scope "$applicationGatewayGroupId"
    ```

### Using Service Principal
It is also possible to provide AGIC access to ARM via a Kubernetes secret.

 1. Create an Active Directory Service Principal and encode with base64. The base64 encoding is required for the JSON blob to be saved to Kubernetes.
    ```bash
    az ad sp create-for-rbac --sdk-auth > auth.json
    spBase64Encoded=$(cat auth.json | base64 -w0)
    spAppId=$(jq -r ".appId" auth.json)
    ```

1. Assign `Contributor` role to the Application Gateway resource group.
    ```bash
    applicationGatewayGroupId=$(az group show -g $applicationGatewayGroupName -o tsv --query "id")

    az role assignment create \
            --role Contributor \
            --assignee "$spAppId" \
            --scope "$applicationGatewayGroupId"
    ```

## Set up Application Gateway Ingress Controller

With the instructions in the previous section we created and configured a new AKS cluster and
an App Gateway. We are now ready to deploy a sample app and an ingress controller to our new
Kubernetes infrastructure.

<details>
<summary><strong>Install Helm (skip if already installed)</strong></summary>

[Helm](https://docs.microsoft.com/en-us/azure/aks/kubernetes-helm) is a package manager for
Kubernetes. We will leverage it to install the `application-gateway-kubernetes-ingress` package:
- *RBAC enabled* AKS cluster
    ```bash
    kubectl create serviceaccount --namespace kube-system tiller-sa
    kubectl create clusterrolebinding tiller-cluster-rule --clusterrole=cluster-admin --serviceaccount=kube-system:tiller-sa
    helm init --tiller-namespace kube-system --service-account tiller-sa
    ```

- *RBAC disabled* AKS cluster
    ```bash
    helm init
    ```
</details>

<details>
<summary><strong>Install AAD Pod Identity (skip if already installed or using service principal for authentication)</strong></summary>
Azure Active Directory Pod Identity provides token-based access to [Azure Resource Manager (ARM)](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-overview).

[AAD Pod Identity](https://github.com/Azure/aad-pod-identity) will add the following components to your Kubernetes cluster:
1. Kubernetes [CRDs](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/): `AzureIdentity`, `AzureAssignedIdentity`, `AzureIdentityBinding`
1. [Managed Identity Controller (MIC)](https://github.com/Azure/aad-pod-identity#managed-identity-controllermic) component
1. [Node Managed Identity (NMI)](https://github.com/Azure/aad-pod-identity#node-managed-identitynmi) component

To install AAD Pod Identity to your cluster:
```bash
helm repo add add-pod-identity https://raw.githubusercontent.com/Azure/aad-pod-identity/master/charts
helm repo update
helm install add-pod-identity/aad-pod-identity --set rbac.enabled=true # false if RBAC is disabled on cluster (default is enabled)
```
</details>

### Install Ingress Controller Helm Chart

1. Add the AGIC Helm repository
    ```bash
    helm repo add application-gateway-kubernetes-ingress https://appgwingress.blob.core.windows.net/ingress-azure-helm-package/
    helm repo update
    ```

1. Install AGIC using helm. You can provide additional [helm options](../helm) during installation or you can also create a [helm-config.yaml](../examples/sample-helm-config.yaml).
    > **Note**: It is really important to set `appgw.shared=true` if you want to preserve the existing request rules on Application Gateway. If not, then set appgw.shared=false. Please refer to [Multi-cluster / Shared App Gateway documentation](../features/shared-app-gateway) to understand how sharing of Application Gateway works.

    ```bash
    applicationGatewayId=$(az network application-gateway show -n $applicationGatewayName -g $applicationGatewayGroupName -o tsv --query "id")

    # Using User assigned identity
    helm install application-gateway-kubernetes-ingress/ingress-azure \
      --set appgw.applicationGatewayID=$applicationGatewayId \
      --set appgw.shared=false \
      --set armAuth.type=aadPodIdentity \
      --set armAuth.identityResourceID=$identityResourceId \
      --set armAuth.identityClientID=$identityClientId \
      --set rbac.enabled=true \ # false if RBAC is disabled on cluster (default is enabled)
      --version 0.10.0-rc5

    # Using Service principal
    # helm install application-gateway-kubernetes-ingress/ingress-azure \
    #   --set appgw.applicationGatewayID=$applicationGatewayId \
    #   --set appgw.shared=false \
    #   --set armAuth.type=servicePrincipal \
    #   --set armAuth.secretJSON=$spBase64Encoded \
    #   --set rbac.enabled=true \ # false if RBAC is disabled on cluster (default is enabled)
    #   --version 0.10.0-rc5
    ```

### Install a Sample App
Now that we have App Gateway, AKS, and AGIC installed we can install a sample app
via [Azure Cloud Shell](https://shell.azure.com/):

> **Note**: If you selected `appgw.shared=true` during the helm install step, Please refer to [Multi-cluster / Shared App Gateway documentation](../features/shared-app-gateway) to understand how sharing of Application Gateway works. You need to configure `AzureIngressProhibitedTarget`, otherwise, your ingress will not be exposed by AGIC.

```yaml
cat <<EOF | kubectl apply -f -
apiVersion: v1
kind: Pod
metadata:
  name: aspnetapp
  labels:
    app: aspnetapp
spec:
  containers:
  - image: "mcr.microsoft.com/dotnet/core/samples:aspnetapp"
    name: aspnetapp-image
    ports:
    - containerPort: 80
      protocol: TCP

---

apiVersion: v1
kind: Service
metadata:
  name: aspnetapp
spec:
  selector:
    app: aspnetapp
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80

---

apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: aspnetapp
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
spec:
  rules:
  - http:
      paths:
      - path: /
        backend:
          serviceName: aspnetapp
          servicePort: 80
EOF
```

Alternatively you can:

1. Download the YAML file above:
```bash
curl https://raw.githubusercontent.com/Azure/application-gateway-kubernetes-ingress/master/docs/examples/aspnetapp.yaml -o aspnetapp.yaml
```

2. Apply the YAML file:
```bash
kubectl apply -f aspnetapp.yaml
```


## Other Examples
The **[tutorials](../tutorial.md)** document contains more examples on how toexpose an AKS
service via HTTP or HTTPS, to the Internet with App Gateway.
