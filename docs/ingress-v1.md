# Ingress V1 Support

This document describes AGIC's implementation of specific Ingress resource fields and features.
As the Ingress specification has evolved between v1beta1 and v1, any differences between versions are highlighted to ensure clarity for AGIC users.

**Note: Ingress/V1 is fully supported with AGIC >= 1.5.1**

## Kubernetes Versions
For Kubernetes version 1.19+, the API server translates any Ingress v1beta1 resources to Ingress v1 and AGIC watches Ingress v1 resources.

## IngressClass and IngressClass Name

AGIC now supports using `ingressClassName` property along with `kubernetes.io/ingress.class: azure/application-gateway` to indicate that a specific ingress should processed by AGIC.

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: shopping-app
spec:
  ingressClassName: azure-application-gateway
  ...
```

## Ingress Rules

### Wildcard Hostnames
AGIC supports [wildcard hostnames](https://kubernetes.io/docs/concepts/services-networking/ingress/#hostname-wildcards) as documented by the upstream API as well as precise hostnames.
* Wildcard hostnames are limited to the whole first DNS label of the hostname, e.g. `*.foo.com` is valid but `*foo.com`, `foo*.com`, `foo.*.com` are not.
`*` is also not a valid hostname.

### PathType property (ingress.rules.http.paths.pathType)
> **Breaking Change Notice**
> * PathType property is now a required property in Ingress/V1. Before AGIC 1.5.1, path match types were ignored and path matching was performed with a AGIC/AppGW-specific implementation.
> * Paths prefixed with `*` were treated as Prefix and without were treated as Exact.
> * This behavior is preserved in the `ImplementationSpecific` match type in AGIC 1.5.1+ to ensure backwards compatibility.

AGIC now supports [PathType](https://kubernetes.io/docs/concepts/services-networking/ingress/#path-types) in Ingress V1.
* `Exact` path matches will now result in matching requests to the given path exactly.
* `Prefix` patch match type will now result in matching requests with a "segment prefix" rather than a "string prefix" according to the spec (e.g. the prefix `/foo/bar` will match requests with paths `/foo/bar`, `/foo/bar/`, and `/foo/bar/baz`, but not `/foo/barbaz`).
* `ImplementationSpecific` patch match type preserves the old path behaviour of AGIC < 1.5.1 and allows to backwards compatibility.