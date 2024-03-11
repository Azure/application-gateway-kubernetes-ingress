# How to deploy AGIC via Helm using Workload Identity

This assumes you have an existing Application Gateway. If not, you can create it with command:
## 1. Add the AGIC Helm repository

```bash
helm repo add application-gateway-kubernetes-ingress https://appgwingress.blob.core.windows.net/ingress-azure-helm-package/
helm repo update
```

## 2. Set environment variables

```bash
$APP_GW_ID=""
$AKS_CLUSTER_NAME="aks"
$RESOURCE_GROUP="test"
$USER_ASSIGNED_IDENTITY_NAME="agic-identity"
$FEDERATED_IDENTITY_CREDENTIAL_NAME="agic-identity"
```

## 3. Enable workload identity on AKS and get OIDC profile

```bash
az aks update -g "$RESOURCE_GROUP" -n "$AKS_CLUSTER_NAME" --enable-oidc-issuer --enable-workload-identity

$AKS_OIDC_ISSUER="$(az aks show -n "$AKS_CLUSTER_NAME" -g "$RESOURCE_GROUP" --query "oidcIssuerProfile.issuerUrl" -otsv)"
```

## 4. Create federated identity credential. 

**Note**: the name of the service account that gets created after the helm installation is “ingress-azure” and the following command assumes it will be deployed in “default” namespace. Please change the namespace name in the next command if you deploy the AGIC related Kubernetes resources in other namespace.

```bash
az identity create --name "$USER_ASSIGNED_IDENTITY_NAME" --resource-group "$RESOURCE_GROUP" 

az identity federated-credential create --name "$FEDERATED_IDENTITY_CREDENTIAL_NAME" --identity-name "$USER_ASSIGNED_IDENTITY_NAME" --resource-group "$RESOURCE_GROUP" --issuer "$AKS_OIDC_ISSUER" --subject system:serviceaccount:default:ingress-azure
```

## 5. Obtain the ClientID of the identity created before that is needed for the next step

```bash
$CLIENT_ID=$(az identity show --resource-group "$RESOURCE_GROUP" --name "$USER_ASSIGNED_IDENTITY_NAME" --query 'clientId' -otsv)
```

## 6. Add Contributor role for the identity over the Application Gateway

```bash
az role assignment create --assignee "$CLIENT_ID" --scope "$APP_GW_ID" --role Contributor
```

## 7. Create following file:

```bash
cat <<EOT > helm-config.yaml
appgw:
    applicationGatewayID: "$APP_GW_ID"
rbac:
    enabled: true
armAuth:
    type: workloadIdentity
    identityClientID: "$CLIENT_ID"
EOT
```

## 8.Get the AKS cluster credentials.

```bash
az aks get-credentials -g "$RESOURCE_GROUP" -n "$AKS_CLUSTER_NAME"
```

## 9. Install the helm chart

```bash
helm install ingress-azure \
  -f helm-config.yaml \
  application-gateway-kubernetes-ingress/ingress-azure \
  --version 1.7.3
```
