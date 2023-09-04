- [How to try](#how-to-try)
- [v1.6.0](#v160)
- [v1.6.0-rc1](#v160-rc1)
  - [Features](#features)
  - [Fixes](#fixes)

# v1.6.0
Same as 1.6.0-rc1

# v1.6.0-rc1

## Features
* [#1399](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1399) Support to configure rewrite rule set via CRDs
* [#1370](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1370) Support for selecting SSL Profile using annotation on the HTTPS listener
* [#1377](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1377) Ability to modify container securityContext and additional volumes in Helm

## Fixes
* [#1433](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1433) Drop duplicate hostnames when setting listener hostnames
* [#1432](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1432) Fixes ingress class matching logic
* [#1401](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1401) Fix root cause for issue related to HTTPS being removed when AGIC pod is restarted
* [#1435](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1435) Limit AAD token retries to 10 retries (with 10 second interval) and exit Pod to inform customer of misconfiguration	
* [#1441](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1441) Assume backend protocol as HTTPS when container port is 443 (already assumed for service port 443)

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
  --version 1.6.0-rc1

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
  --version 1.6.0-rc1
```

