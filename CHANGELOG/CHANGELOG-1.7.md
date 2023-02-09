- [How to try](#how-to-try)
- [v1.7.0-rc1](#v170-rc1)
  - [Features](#features)
  - [Fixes](#fixes)

# v1.7.0-rc1

## Features
* [#1498](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1498) Support for Workload Identity in AGIC Helm installation
* [#1503](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1503) Support for Private only Application Gateway
* [#1343](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1343) Support for rule-priority annotation

## Fixes
* [#1497](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1497) Rewrite url config should be empty if not specified in the custom resource
* [#1500](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1500) Increase ARM polling duration to safeguard against transient failures

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
  --version 1.7.0-rc1

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
  --version 1.7.0-rc1
```

