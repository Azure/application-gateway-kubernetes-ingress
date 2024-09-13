## Prerequisites

> [Application Gateway for Containers](https://aka.ms/agc) has been released, which introduces numerous performance, resilience, and feature changes. Please consider leveraging Application Gateway for Containers for your next deployment.

> AGIC charts have been moved to MCR. Use oci://mcr.microsoft.com/azure-application-gateway/charts/ingress-azure as the target repository.

You need to complete the following tasks prior to deploying AGIC on your cluster:

1. Prepare your Azure subscription and your `az-cli` client.

    ```bash
    # Sign in to your Azure subscription.
    SUBSCRIPTION_ID='<your subscription id>'
    az login
    az account set --subscription $SUBSCRIPTION_ID

    # Register required resource providers on Azure.
    az provider register --namespace Microsoft.ContainerService
    az provider register --namespace Microsoft.Network
    ```

2. Set an AKS cluster for your workload.

    > AKS cluster should have the workload identity feature enabled. [Learn how](../../aks/workload-identity-deploy-cluster.md#update-an-existing-aks-cluster) to enable workload identity on an existing AKS cluster.

    If using an existing cluster, ensure you enable Workload Identity support on your AKS cluster.  Workload identities can be enabled via the following:

    ```bash
    AKS_NAME='<your cluster name>'
    RESOURCE_GROUP='<your resource group name>'

    az aks update -g $RESOURCE_GROUP -n $AKS_NAME --enable-oidc-issuer --enable-workload-identity --no-wait
    ```

    If you don't have an existing cluster, use the following commands to create a new AKS cluster and workload identity enabled.

    ```bash
    AKS_NAME='<your cluster name>'
    RESOURCE_GROUP='<your resource group name>'
    LOCATION='northeurope'
    VM_SIZE='<the size of the vm in AKS>' # The size needs to be available in your location

    az group create --name $RESOURCE_GROUP --location $LOCATION
    az aks create \
        --resource-group $RESOURCE_GROUP \
        --name $AKS_NAME \
        --location $LOCATION \
        --node-vm-size $VM_SIZE \
        --network-plugin azure \
        --enable-oidc-issuer \
        --enable-workload-identity \
        --generate-ssh-key
    ```

3. Install Helm

    [Helm](https://github.com/helm/helm) is an open-source packaging tool that is used to install AGIC.

    > Helm is already available in Azure Cloud Shell.  If you are using Azure Cloud Shell, no additional Helm installation is necessary.

    ```bash
    curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash
    ```

## Deploy or Use existing Application Gateway

If using an existing Application Gateway, make sure the following:

1. Set the environment variable.

  ```bash
  APPGW_ID="<existing app gateway resource id>"
  ```

1. [Follow steps](../how-tos/networking.md) here to make sure AppGW VNET is correctly setup i.e. either it is using same VNET as AKS or is peered.

  If you don't have an existing Application Gateway, use the following commands to create a new one.

1. Setup environment variables

    ```bash
    AKS_NAME='<your cluster name>'
    RESOURCE_GROUP='<your resource group name>'
    LOCATION="<your cluster location>"

    APPGW_NAME="application-gateway"
    APPGW_SUBNET_NAME="appgw-subnet"
    ```

1. Deploy subnet for Application Gateway

    ```bash
    nodeResourceGroup=$(az aks show -n $AKS_NAME -g $RESOURCE_GROUP -o tsv --query "nodeResourceGroup")
    aksVnetName=$(az network vnet list -g $nodeResourceGroup -o tsv --query "[0].name")
    aksVnetId=$(az network vnet show -n $aksVnetName -g $nodeResourceGroup -o tsv --query "id")
    az network vnet subnet create \
        --resource-group $nodeResourceGroup  \
        --vnet-name $aksVnetName \
        --name $APPGW_SUBNET_NAME \
        --address-prefixes "10.226.0.0/23"

    APPGW_SUBNET_ID=$(az network vnet subnet list --resource-group $nodeResourceGroup --vnet-name $aksVnetName --query "[?name=='$APPGW_SUBNET_NAME'].id" --output tsv)
    ```

1. Deploy Application Gateway

    ```bash
    az network application-gateway create \
      --name $APPGW_NAME \
      --location $LOCATION \
      --resource-group $RESOURCE_GROUP \
      --subnet $APPGW_SUBNET_ID \
      --capacity 2 \
      --sku Standard_v2 \
      --http-settings-cookie-based-affinity Disabled \
      --frontend-port 80 \
      --http-settings-port 80 \
      --http-settings-protocol Http \
      --public-ip-address appgw-ip \
      --priority 10
    
    APPGW_ID=$(az network application-gateway show --name $APPGW_NAME --resource-group $RESOURCE_GROUP --query "id" --output tsv)
    ```

## Install Application Gateway Ingress Controller

1. Setup environment variables

    ```bash
    AKS_NAME='<your cluster name>'
    RESOURCE_GROUP='<your resource group name>'
    LOCATION="<your cluster location>"

    IDENTITY_RESOURCE_NAME='agic-identity'
    ```

1. Create a user managed identity for AGIC controller and federate the identity as Workload Identity to use in the AKS cluster.

    ```bash
    echo "Creating identity $IDENTITY_RESOURCE_NAME in resource group $RESOURCE_GROUP"
    az identity create --resource-group $RESOURCE_GROUP --name $IDENTITY_RESOURCE_NAME
    IDENTITY_PRINCIPAL_ID="$(az identity show -g $RESOURCE_GROUP -n $IDENTITY_RESOURCE_NAME --query principalId -otsv)"
    IDENTITY_CLIENT_ID="$(az identity show -g $RESOURCE_GROUP -n $IDENTITY_RESOURCE_NAME --query clientId -otsv)"

    echo "Waiting 60 seconds to allow for replication of the identity..."
    sleep 60

    echo "Set up federation with AKS OIDC issuer"
    AKS_OIDC_ISSUER="$(az aks show -n "$AKS_NAME" -g "$RESOURCE_GROUP" --query "oidcIssuerProfile.issuerUrl" -o tsv)"
    az identity federated-credential create --name "agic" \
        --identity-name "$IDENTITY_RESOURCE_NAME" \
        --resource-group $RESOURCE_GROUP \
        --issuer "$AKS_OIDC_ISSUER" \
        --subject "system:serviceaccount:default:ingress-azure"

    resourceGroupId=$(az group show --name $RESOURCE_GROUP --query id -otsv)
    nodeResourceGroup=$(az aks show -n $AKS_NAME -g $RESOURCE_GROUP -o tsv --query "nodeResourceGroup")
    nodeResourceGroupId=$(az group show --name $nodeResourceGroup --query id -otsv)

    echo "Apply role assignments to AGIC identity"
    az role assignment create --assignee-object-id $IDENTITY_PRINCIPAL_ID --assignee-principal-type ServicePrincipal --scope $resourceGroupId --role "Reader"
    az role assignment create --assignee-object-id $IDENTITY_PRINCIPAL_ID --assignee-principal-type ServicePrincipal --scope $nodeResourceGroupId --role "Contributor"
    az role assignment create --assignee-object-id $IDENTITY_PRINCIPAL_ID --assignee-principal-type ServicePrincipal --scope $APPGW_ID --role "Contributor"
    ```

   > Assignment of the managed identity immediately after creation may result in an error that the principalId does not exist. Allow about a minute of time to elapse for the identity to replicate in Microsoft Entra ID prior to delegating the identity.

1. Install AGIC using Helm

### For new deployments

AGIC can be installed by running the following commands:

  ```bash
  az aks get-credentials --resource-group $RESOURCE_GROUP --name $AKS_NAME

  # on aks cluster with only linux node pools
  helm install ingress-azure \
    oci://mcr.microsoft.com/azure-application-gateway/charts/ingress-azure \
    --set appgw.applicationGatewayID=$APPGW_ID \
    --set armAuth.type=workloadIdentity \
    --set armAuth.identityClientID=$IDENTITY_CLIENT_ID \
    --set rbac.enabled=true \
    --version 1.7.3
  
  # on aks cluster with windows node pools
  helm install ingress-azure \
    oci://mcr.microsoft.com/azure-application-gateway/charts/ingress-azure \
    --set appgw.applicationGatewayID=$APPGW_ID \
    --set armAuth.type=workloadIdentity \
    --set armAuth.identityClientID=$IDENTITY_CLIENT_ID \
    --set rbac.enabled=true \
    --set nodeSelector."beta\.kubernetes\.io/os"=linux \
    --version 1.7.3
  ```

### For existing deployments

AGIC can be upgraded by running the following commands:

  ```bash
  az aks get-credentials --resource-group $RESOURCE_GROUP --name $AKS_NAME

  # on aks cluster with only linux node pools
  helm upgrade ingress-azure \
    oci://mcr.microsoft.com/azure-application-gateway/charts/ingress-azure \
    --set appgw.applicationGatewayID=$APPGW_ID \
    --set armAuth.type=workloadIdentity \
    --set armAuth.identityClientID=$IDENTITY_CLIENT_ID \
    --set rbac.enabled=true \
    --version 1.7.3
  
  # on aks cluster with windows node pools
  helm upgrade ingress-azure \
    oci://mcr.microsoft.com/azure-application-gateway/charts/ingress-azure \
    --set appgw.applicationGatewayID=$APPGW_ID \
    --set armAuth.type=workloadIdentity \
    --set armAuth.identityClientID=$IDENTITY_CLIENT_ID \
    --set rbac.enabled=true \
    --set nodeSelector."beta\.kubernetes\.io/os"=linux \
    --version 1.7.3
  ```

### Install a Sample App

Now that we have App Gateway, AKS, and AGIC installed we can install a sample app
via [Azure Cloud Shell](https://shell.azure.com/):

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
    - image: "mcr.microsoft.com/dotnet/samples:aspnetapp"
      name: aspnetapp-image
      ports:
      - containerPort: 8080
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
      targetPort: 8080
  
  ---
  
  apiVersion: networking.k8s.io/v1
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
            service:
              name: aspnetapp
              port:
                number: 80
          pathType: Exact
  EOF
  ```

Alternatively you can:

1. Download the YAML file above:

    ```bash
    curl https://raw.githubusercontent.com/Azure/application-gateway-kubernetes-ingress/master/docs/examples/aspnetapp.yaml -o aspnetapp.yaml
    ```

1. Apply the YAML file:

    ```bash
    kubectl apply -f aspnetapp.yaml
    ```

## Other Examples

The **[tutorials](../tutorials/tutorial.general.md)** document contains more examples on how to expose an AKS service via HTTP or HTTPS, to the Internet with App Gateway.
