- [How to try](#how-to-try)
- [v1.5.1](#v151)
  - [Features](#features)
  - [Fixes](#fixes)
- [v1.5.0](#v150)
  - [Features](#features)
  - [Fixes](#fixes)
- [v1.5.0-rc1](#v150-rc1)
  - [Features](#features-1)
  - [Fixes](#fixes-1)

# v1.5.1

## Features
*  [[#1122](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1346) Add support for ingress.rules.path.pathType property

## Fixes
* [#1347](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1347) fix(v1/ingress): fix panic due to ingress class when k8s <= 1.19
* [#1344](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1344) fix(v1/ingress): retry getting server version to get past transient issues

# v1.5.0

## Features
* [#1329](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1329) Support for ingress class resource in v1/ingress
* [#1280](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1280) Add cookie-based-affinity-distinct-name annotation
* [#1287](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1287) Add rewrite rule set annotation to reference an existing rewrite rule on Application Gateway

## Fixes
* [#1322](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1322) fix(port): port reference by name in ingress

# v1.5.0-rc1

## Features
* [#1197](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1197) Support for v1/ingress (maintaining compatibility with v1beta1/ingress for clsuters <= 1.22)
* [#1324](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1324) Support for multi-arch images: amd64/arm64
* [#1169](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1169) Resource quota exposed through helm chart
> Note:
>  * Support for ingress class resource will added in a future release
>  * Support for ingress.rules.path.pathType property will added in a future release

## Fixes
* [#1282](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1282) fix(ingress): Modify cloned ingress instead of original ingress
* [#1271](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1271) fix(private clouds): Use correct endpoints for AAD and Azure in private clouds
* [#1273](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1273) fix(config): panic when processing duplicate paths in urlpathmap
* [#1220](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1220) fix(crd): Upgrade prohibited target CRD api-version and add tests
* [#1278](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/1278) fix(prohibited target): incorrect merge when rules being merged reference the same path map