- [How to try](#how-to-try)
- [v1.4.0-rc1](#v140-rc1)
  - [Features](#features)
  - [Fixes](#fixes)

# v1.4.0-rc1

## Features:
* [#1062](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1062) Add additional annotations for health probe
* [#1064](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1064) Add tolerations and affinities to Helm configuration
* [#1084](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1084) Add custom annotatios on AGIC pod
* [#1130](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1130) Add security context to helm configuration


### Miscellaneuos:
* [#1075](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1075) Moving to newer RBAC Api version
* [#1080](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1080) Printing operation ID in AGIC logs
* [#1081](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1081) Replace glog with klog for logging
* [#1082](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1082) Add a unique identifier in User Agent for ARM requests

## Fixes:
* [#1070](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1070) Update ingressSecretMap even when secret is malformed
* [#1073](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1073) Small adjustment in the values.yaml template to generate the artifact correctly
* [#1090](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1090) Refactor backend port resolution in Backend Http Settings
* [#1123](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1123) Remove tag LastUpdatedByK8sIngress
* [#1125](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1125) Generate default listener with private ip when specified in helm


## How to try:
```bash
# upgrade to the latest release version 1.4.0-rc1
helm repo update
helm install \
  <release-name> \
  -f helm-config.yaml \
  application-gateway-kubernetes-ingress/ingress-azure

# use --version 1.4.0-rc1 when installing/upgrading using helm
helm repo update
helm install \
  <release-name> \
  -f helm-config.yaml \
  application-gateway-kubernetes-ingress/ingress-azure \
  --version 1.4.0-rc1

# or 

# https://github.com/Azure/application-gateway-kubernetes-ingress/blob/master/docs/how-tos/helm-upgrade.md
# --reuse-values   when upgrading, reuse the last release's values and merge in any overrides from the command line via --set and -f. If '--reset-values' is specified, this is ignored
helm repo update
helm upgrade \
  <release-name> \
  application-gateway-kubernetes-ingress/ingress-azure \
  --reuse-values \
  --version 1.4.0-rc1
```
