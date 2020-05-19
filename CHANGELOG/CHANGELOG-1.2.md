- [v1.2.0-rc2](#v120-rc2)
  - [Important Note](#important-note)
  - [Fixes](#fixes)
- [v1.2.0-rc1](#v120-rc1)
  - [Features](#features)
  - [Fixes](#fixes-1)
  - [Known Issues](#known-issues)

# v1.2.0-rc2

#### Important Note

In this release, AGIC will use the new `hostnames` property in HTTP Listener in Application Gateway instead of `hostname`. With this property, We will be able to expose support for Wild Card hostnames with characters like * and ? allowed to match characters.  
We are working on bringing Azure Portal support for the new property soon. Until those changes arrive, Users will not be able to view the hostname in the listener section on Portal.

## Fixes:
* [#828](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/828): Address breaking change in AAD Pod Identity v1.6 related to case-sensitvity in Azure Identity CRD.
* [#779](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/779), [#752](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/752), [#635](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/635), [#629](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/629): Fix nil pointer dereference exception when an ingress is not having HTTP ingress rule.
* [#851](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/851): When service port is 443, automatically default to using https protocol for backend http setting and health probe.
* [#766](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/766): Apply WAF policy to listener when path is any of "/", "/*", "". This is in regards to the usage of `waf-policy-for-path` annotation.
* [#850](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/850): helm: update deployment when config changes by using a checksum.
* Switch from `hostname` to `hostnames` property in HTTP Listener on Application Gateway.
* helm: remove replica count setting from supported helm values. This will be added back when AGIC will recieve support for leader election.

# v1.2.0-rc1

## Features:

* New Annotations added:
  * [#765](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/765): `appgw.ingress.kubernetes.io/appgw-ssl-certificate`:  Allow using existing ssl certificates on Application Gateway with a listener
  * [#776](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/776): `appgw.ingress.kubernetes.io/appgw-trusted-root-certificate`:  Allow using existing trusted root certificates on Application Gateway when using https using a self-signed certificate on the backend.
  * [#701](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/701): `appgw.ingress.kubernetes.io/backend-hostname`: Ability to provide a custom host name for connecting to the pods.
* [#775](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/775): Reconcile(**beta**) - If provided with `reconcilePeriodSeconds`, AGIC will reconcile the gateway every `reconcilePeriodSeconds` to bring the gateway back to expected state. This feature is in **beta** and may have issues.
* [#723](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/723): When using kubenet with AKS, AGIC will now try to automatically assign the node pool's route table to Application Gateway's subnet subjected to AGIC having needed permissions. Route table assignment is needed to setup connectivity between Application Gateway and Pods. If this step fails, you can resolve this by manually performing the assignment.
* Updated AGIC's `deployment` object's api version from `apps/v1beta2` to `apps/v1` to support k8s 1.16. When upgrading an existing helm release, you will see **conflict** when helm tries to update the deployment object.

## Fixes:
* [#789](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/789): Even though the bug has been fixed, currently, setting a different protocol on Probe than HTTP setting will cause cause a validation error.
* [#686](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/686): Skip updating the gateway when in non-operational state

## Known Issues:
* Issues with installation of ProhibitedTarget CRD using Helm 3. Helm 3 has done away with `crd-install` hooks and requires CRDs to moved to a separate `crd` folder.
* When upgrading an existing helm release, you will see **conflict** when helm tries to update the deployment object.
  ```bash
  Error: UPGRADE FAILED: rendered manifests contain a new resource that already exists. Unable to continue with update: existing resource conflict: namespace: default, name: <release-name>, existing_kind: apps/v1, Kind=Deployment, new_kind: apps/v1, Kind=Deployment
  ```

## How to try:
```bash
# use --version 1.2.0-rc2 when installing/upgrading using helm
helm install \
  <release-name> \
  -f helm-config.yaml \
  application-gateway-kubernetes-ingress/ingress-azure \
  --version 1.2.0-rc2

# or 

# https://github.com/Azure/application-gateway-kubernetes-ingress/blob/master/docs/how-tos/helm-upgrade.md
helm upgrade \
  <release-name> \
  application-gateway-kubernetes-ingress/ingress-azure \
  --version 1.2.0-rc2
```