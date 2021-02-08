- [How to try](#how-to-try)
- [v1.4.0-rc1](#v140-rc1)
  - [Features](#features)
  - [Fixes](#fixes)

# v1.4.0-rc1

## Features:
* [#1062](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1062) Add ingress annotations to customize health probe
    * `appgw.ingress.kubernetes.io/health-probe-hostname`
    * `appgw.ingress.kubernetes.io/health-probe-port`
    * `appgw.ingress.kubernetes.io/health-probe-path`
    * `appgw.ingress.kubernetes.io/health-probe-status-codes`
    * `appgw.ingress.kubernetes.io/health-probe-interval`
    * `appgw.ingress.kubernetes.io/health-probe-timeout`
    * `appgw.ingress.kubernetes.io/health-probe-unhealthy-threshold`
* [#1064](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1064) Allow modifying tolerations and affinities through Helm configuration to AGIC pod
* [#1084](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1084) Allow adding custom annotatios through helm configuration to AGIC pod
* [#1130](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1130) Allow modifying security context through helm configuration on AGIC pod

### Miscellaneuos:
* [#1075](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1075) Moving to RBAC Api version `rbac.authorization.k8s.io/v1`
* [#1080](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1080) Log ARM operation ID in AGIC logs after PUT operation on Application Gateway
* [#1081](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1081) Replace glog with klog for logging
* [#1082](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1082) Add a unique identifier in User Agent for ARM requests

## Fixes:
* [#1070](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1070) Fix secret and ingress handlers to handle case where secret is created as empty and repopulated later
* [#1073](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1073) Small adjustment in the values.yaml template to generate the artifact correctly
* [#1090](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1090) Fix backend pool processing to `continue` instead of `break` when an error condition is encountered in services
* [#1123](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1123) Remove tag LastUpdatedByK8sIngress
* [#1125](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1125) Generate default listener with private ip when specified in helm


## How to try:
```bash
# Add helm repo / update AGIC repo
helm repo add application-gateway-kubernetes-ingress https://appgwingress.blob.core.windows.net/ingress-azure-helm-package/
helm repo update

# Install
# use --version 1.4.0-rc1 when installing using helm
helm install \
  <release-name> \
  -f helm-config.yaml \
  application-gateway-kubernetes-ingress/ingress-azure \
  --version 1.4.0-rc1

# or 

# Upgrade
# https://github.com/Azure/application-gateway-kubernetes-ingress/blob/master/docs/how-tos/helm-upgrade.md
# --reuse-values   when upgrading, reuse the last release's values and merge in any overrides from the command line via --set and -f. If '--reset-values' is specified, this is ignored
helm upgrade \
  <release-name> \
  application-gateway-kubernetes-ingress/ingress-azure \
  --reuse-values \
  --version 1.4.0-rc1
```
