- [How to try](#how-to-try)
- [v1.9.0](#v170-rc1)
  - [Features](#features)
  - [Fixes](#fixes)

# v1.9.0

## Features
* [#1703](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1703) Application Gateway subnet delegation support

## How to try:
```bash
# Add helm repo / update AGIC repo
helm repo add application-gateway-kubernetes-ingress https://appgwingress.blob.core.windows.net/ingress-azure-helm-package/
helm repo update

# Install
helm install \
  <release-name> \
  -f helm-config.yaml \
  application-gateway-kubernetes-ingress/ingress-azure \
  --version 1.9.0

# or

# Upgrade
# https://github.com/Azure/application-gateway-kubernetes-ingress/blob/master/docs/how-tos/helm-upgrade.md
# --reuse-values   when upgrading, reuse the last release's values and merge in any overrides from the command line via --set and -f. If '--reset-values' is specified, this is ignored

# Install CRDs separately as helm upgrade doesn't install CRDs.
kubectl apply -f https://raw.githubusercontent.com/Azure/application-gateway-kubernetes-ingress/master/helm/ingress-azure/crds/azureapplicationgatewayrewrite.yaml

helm upgrade \
  <release-name> \
  application-gateway-kubernetes-ingress/ingress-azure \
  --reuse-values
  --version 1.9.0
```

