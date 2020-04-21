#!/bin/bash

buildid="$1"

[[ -z "$buildid" ]] && (echo "buildid is not set"; exit 1)

echo "Installing BuildId $buildid"

kubectl delete namespace/agic
kubectl create namespace agic

helm repo add staging https://appgwingress.blob.core.windows.net/ingress-azure-helm-package-staging/
helm repo update

helm install agic-$buildid staging/ingress-azure \
  --set appgw.applicationGatewayID=${applicationGatewayId} \
  --set armAuth.type=aadPodIdentity \
  --set armAuth.identityResourceID=${identityResourceId} \
  --set armAuth.identityClientID=${identityClientId} \
  --set rbac.enabled=true \
  -n agic \
  --version $buildid
