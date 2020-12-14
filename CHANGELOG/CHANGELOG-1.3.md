- [How to try](#how-to-try)
- [v1.3.0](#v130)
- [v1.3.0-rc1](#v130-rc1)
  - [Features](#features)
  - [Fixes](#fixes)

# v1.3.0
Same as v1.3.0-rc1

# v1.3.0-rc1

## Features:
* [#974](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/974) Support custom listener port using override-frontend-port annotation
* [#801](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/801) Support custom ingress class
* [#917](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/917) Support for configuring prohibited targets through helm config
* [#1008](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/1008) Ability to provide SKU when deploying gateway though AGIC
* [#958](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/958) Support sub-resource name prefix

### Miscellaneuos:
* [#1027](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/1027) Updated to Ubuntu 20.04 LTS
* [#918](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/918) Update azure go sdk version to 2020-05-01

## Fixes:
* [#990](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/990) Choose backend port in case backend port name is resolved to multiple backend ports
* [#975](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/975) Duplicate URL Path Map names w/ certain Ingress YAML
* [#991](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/991) Merge path rules such that duplicate paths are ignored
* [#1018](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/1018) Add security context to have permission to azure.json

## How to try:
```bash
# upgrade to the latest release version 1.3.0
helm repo update
helm install \
  <release-name> \
  -f helm-config.yaml \
  application-gateway-kubernetes-ingress/ingress-azure

# use --version 1.3.0-rc1 when installing/upgrading using helm
helm repo update
helm install \
  <release-name> \
  -f helm-config.yaml \
  application-gateway-kubernetes-ingress/ingress-azure \
  --version 1.3.0-rc1

# or 

# https://github.com/Azure/application-gateway-kubernetes-ingress/blob/master/docs/how-tos/helm-upgrade.md
# --reuse-values   when upgrading, reuse the last release's values and merge in any overrides from the command line via --set and -f. If '--reset-values' is specified, this is ignored
helm repo update
helm upgrade \
  <release-name> \
  application-gateway-kubernetes-ingress/ingress-azure \
  --reuse-values \
  --version 1.3.0-rc1
```
