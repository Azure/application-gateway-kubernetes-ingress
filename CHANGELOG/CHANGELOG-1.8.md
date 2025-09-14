- [How to try](#how-to-try)
- [v1.8.1](#v181)
  - [Features](#features)
  - [Fixes](#fixes)

# v1.8.1

## Fixes
* [#1708](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1708) Bumped go dependency versions
* [#1707](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1707) Fixed Overlay CNI label checking

# v1.8.0

## Features
* [#1650](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1650) Support for Overlay CNI

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
  --version 1.8.1

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
  --version 1.8.1
```
