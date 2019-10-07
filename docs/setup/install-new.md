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

With the instructions in the previous section we created and configured a new AKS cluster and
an App Gateway. We are now ready to deploy an sample app and an ingress controller to our new
Kubernetes infrastructure.

### Setup Kubernetes Credentials
For the following steps we need setup [kubectl](https://kubectl.docs.kubernetes.io/) command,
which we will use to connect to our new Kubernetes cluster. [Cloud Shell](https://shell.azure.com/) has `kubectl` already installed. We will use `az` CLI to obtain credentials for Kubernetes.

Get credentials for your newly deployed AKS ([read more](https://docs.microsoft.com/en-us/azure/aks/kubernetes-walkthrough#connect-to-the-cluster)):
    ```bash
    az aks get-credentials --resource-group <your-new-resource-group> --name <name-of-new-AKS-cluster>
    ```

### Install AAD Pod Identity
  Azure Active Directory Pod Identity provides token-based access to
  [Azure Resource Manager (ARM)](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-overview).

  [AAD Pod Identity](https://github.com/Azure/aad-pod-identity) will add the following components to your Kubernetes cluster:
   1. Kubernetes [CRDs](https://kubernetes.io/docs/tasks/access-kubernetes-api/custom-resources/custom-resource-definitions/): `AzureIdentity`, `AzureAssignedIdentity`, `AzureIdentityBinding`
   1. [Managed Identity Controller (MIC)](https://github.com/Azure/aad-pod-identity#managed-identity-controllermic) component
   1. [Node Managed Identity (NMI)](https://github.com/Azure/aad-pod-identity#node-managed-identitynmi) component


  To install AAD Pod Identity to your cluster:

   - *RBAC enabled* AKS cluster

  ```bash
  kubectl create -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml
  ```

   - *RBAC disabled* AKS cluster

  ```bash
  kubectl create -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment.yaml
  ```

### Install Helm
[Helm](https://docs.microsoft.com/en-us/azure/aks/kubernetes-helm) is a package manager for
Kubernetes. We will leverage it to install the `application-gateway-kubernetes-ingress` package:

1. Install [Helm](https://docs.microsoft.com/en-us/azure/aks/kubernetes-helm) and run the following to add `application-gateway-kubernetes-ingress` helm package:

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

1. Add the AGIC Helm repository:
    ```bash
    helm repo add application-gateway-kubernetes-ingress https://appgwingress.blob.core.windows.net/ingress-azure-helm-package/
    helm repo update
    ```

### Install Ingress Controller Helm Chart

1. Download [helm-config.yaml](../examples/sample-helm-config.yaml), which will be used to configure AGIC:
    ```bash
    wget https://raw.githubusercontent.com/Azure/application-gateway-kubernetes-ingress/master/docs/examples/sample-helm-config.yaml -O helm-config.yaml
    ```

1. Edit the newly downloaded [helm-config.yaml](../examples/sample-helm-config.yaml) and fill out the sections `appgw` and `armAuth`.
    ```bash
    nano helm-config.yaml
    ```

   Values:
     - `verbosityLevel`: Sets the verbosity level of the AGIC logging infrastructure. See [Logging Levels](https://github.com/Azure/application-gateway-kubernetes-ingress/blob/463a87213bbc3106af6fce0f4023477216d2ad78/docs/troubleshooting.md#logging-levels) for possible values.
     - `appgw.subscriptionId`: The Azure Subscription ID in which App Gateway resides. Example: `a123b234-a3b4-557d-b2df-a0bc12de1234`
     - `appgw.resourceGroup`: Name of the Azure Resource Group in which App Gateway was created. Example: `app-gw-resource-group`
     - `appgw.name`: Name of the Application Gateway. Example: `applicationgatewayd0f0`
     - `appgw.shared`: This boolean flag should be defaulted to `false`. Set to `true` should you need a [Shared App Gateway](https://github.com/Azure/application-gateway-kubernetes-ingress/blob/072626cb4e37f7b7a1b0c4578c38d1eadc3e8701/docs/setup/install-existing.md#multi-cluster--shared-app-gateway).
     - `kubernetes.watchNamespace`: Specify the name space, which AGIC should watch. This could be a single string value, or a comma-separated list of namespaces.
    - `armAuth.type`: could be `aadPodIdentity` or `servicePrincipal`
    - `armAuth.identityResourceID`: Resource ID of the Azure Managed Identity
    - `armAuth.identityClientId`: The Client ID of the Identity. See below for more information on Identity
    - `armAuth.secretJSON`: Only needed when Service Principal Secret type is chosen (when `armAuth.type` has been set to `servicePrincipal`) 


   Note on Identity: The `identityResourceID` and `identityClientID` are values that were created
   during the [Create an Identity](https://github.com/Azure/application-gateway-kubernetes-ingress/blob/072626cb4e37f7b7a1b0c4578c38d1eadc3e8701/docs/setup/install-new.md#create-an-identity)
   steps, and could be obtained again using the following command:
   ```bash
   az identity show -g <resource-group> -n <identity-name>
   ```
   `<resource-group>` in the command above is the resource group of your App Gateway. `<identity-name>` is the name of the created identity. All identities for a given subscription can be listed using: `az identity list`


1. Install the Application Gateway ingress controller package:

    ```bash
    helm install -f helm-config.yaml application-gateway-kubernetes-ingress/ingress-azure
    ```

### Install a Sample App
Now that we have App Gateway, AKS, and AGIC installed we can install a sample app
via [Azure Cloud Shell](https://shell.azure.com/):

Save the below yaml as `aspnetapp.yaml` and run `kubectl apply -f aspnetapp.yaml`

 ```yaml
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
```

Alternatively you can:

1. Download the YAML file above:
```bash
curl https://raw.githubusercontent.com/Azure/application-gateway-kubernetes-ingress/master/docs/examples/aspnetapp.yaml -o aspnetapp.yaml
```

2. Apply the YAML file:
```bash
kubectl apply -f apsnetapp.yaml
```


## Other Examples
The **[tutorials](../tutorial.md)** document contains more examples on how toexpose an AKS
service via HTTP or HTTPS, to the Internet with App Gateway.
