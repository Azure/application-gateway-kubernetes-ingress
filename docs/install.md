# Install

## Authentication with Azure Resource Manager

The ingress controller supports two ways of authenticating with the Azure Resource Manager

1. Manually create a Azure SDK authentication JSON file and upload it to the AKS cluster as a secret
2. **(Prefered)** Use [AAD-Pod-Identity](https://github.com/Azure/aad-pod-identity), see below

## Get Started

In this example, we will be deploying the ingress controller with `AAD-Pod-Identity` as the authentication with ARM

### Prerequisites

- Created an application gateway and an AKS cluster
- Configured networking of the application gateway and the AKS cluster so that the pods are accessible by the application gateway
- Installed `helm` in the AKS cluster

### Configuring AAD-Pod-Identity

#### Deploy `aad-pod-identity` infra 

Please see https://github.com/Azure/aad-pod-identity#deploy-the-azure-aad-identity-infra for more inromation

```bash
kubectl create -f deploy/infra/deployment.yaml
```

#### Create and Install Azure Identity

1. Create an Azure identity **in the same resource group as the AKS nodes** (typically the resource group with a `MC_` prefix string)

    ```bash
    az identity create -g <resourcegroup> -n <identity-name>
    ```

2. Assign this new identity `Contributor` acess on the application gateway
3. Find the resource and client ID for this identity

    ```bash
    az identity show -g <resourcegroup> -n <identity-name>
    ```

4. Edit [aadpodidentity.yaml](example/aadpodidentity/aadpodidentity.yaml) to include values from the Azure identity

    ```yaml
    # Please see https://github.com/Azure/aad-pod-identity for more inromation
    apiVersion: "aadpodidentity.k8s.io/v1"
    kind: AzureIdentity
    metadata:
      name: azure-identity-appgw-ingress          # will be used by aadpodidbinding.yaml
    spec:
      type: 0
      ResourceID: "/subscriptions/<subscription-id>/resourceGroups/<mc-resourcegroup-name>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<identity-name>"
      ClientID: "<identity-client-id>"
    ```

5. Edit [aadpodidbinding.yaml](example/aadpodidentity/aadpodidbinding.yaml) to match the `aadpodidentity.yaml`

    ```yaml
    # Please see https://github.com/Azure/aad-pod-identity for more inromation
    apiVersion: "aadpodidentity.k8s.io/v1"
    kind: AzureIdentityBinding
    metadata:
      name: azure-identity-binding-appgw-ingress
    spec:
      AzureIdentity: azure-identity-appgw-ingress # metadata.name defined in aadpodidentity.yaml
      Selector: appgw-ingress                     # will be used by helm, identifies the pod using aadpodidbinding label
    ```

6. Apply `aadpodidentity.yaml` and `aadpodidbinding.yaml`

    ```bash
    kubectl apply -f aadpodidentity.yaml
    kubectl apply -f aadpodidbinding.yaml
    ```

    These two resources establishes binding between the Azure identity and the ingress controller pod that we will be creating.

    In this example, the `aad-pod-identity` infra will automatically attach  the Azure identity to the pod identified by label `aadpodidbinding:appgw-ingress`.

### Install Ingress Controller as a Helm Chart

1. Add the `application-gateway-kubernetes-ingress` helm repo and perform a helm update

    ```bash
    helm repo add application-gateway-kubernetes-ingress https://azure.github.io/application-gateway-kubernetes-ingress/helm/
    helm repo update
    ```

2. Edit [helm-config.yaml](example/helm-config.yaml) and fill in the values to match your use case

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
    # Specify the authentication with Azure Resource Manager
    #
    # Two authentication methods are available:
    # - Option 1: AAD-Pod-Identity (https://github.com/Azure/aad-pod-identity)
    armAuth:
      type: aadPodIdentity
      binding: appgw-ingress  # the Selector defined in aadpodidbinding.yaml

    # - Option 2: ServicePrincipal as a kubernetes secret
    # armAuth:
    #   type: servicePrincipal
    #   secretName: networking-appgw-k8s-azure-service-principal
    #   secretKey: ServicePrincipal.json
    ```

3. Install the helm chart `application-gateway-kubernetes-ingress` with the `helm-config.yaml` configuration from the previous step

    ```bash
    helm install -f <helm-config.yaml> application-gateway-kubernetes-ingress
    ```

4. Check the log of the newly created pod to verify if it started properly