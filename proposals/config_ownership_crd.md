# CRD for configuration ownership in brownfield deployments
##### Deploying Ingress Controller to hybrid environments with partial Application Gateway configuration ownership


### Document Purpose
This document is a proposal for the creation of 2 new [CRDs](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) with the goal of
augmenting the configuration of [Azure Application Gateway Ingress Controller](https://azure.github.io/application-gateway-kubernetes-ingress/).

  - Authors: [Akshay Gupta](https://github.com/akshaysngupta), [Delyan Raychev](https://github.com/draychev)
  - Published: June 2nd, 2019
  - Status: Open for comments

### Problem Statement
As of this writing Ingress Controller (v0.6.0) blindly overwrites existing Application Gateway configurations when applying changes to:
  - [listeners](https://github.com/Azure/application-gateway-kubernetes-ingress/blob/9820ff8aed758b2626c068e2fb25629e06540159/pkg/controller/controller.go#L100)
  - [routing rules](https://github.com/Azure/application-gateway-kubernetes-ingress/blob/9820ff8aed758b2626c068e2fb25629e06540159/pkg/controller/controller.go#L107)
  - [HTTP settings](https://github.com/Azure/application-gateway-kubernetes-ingress/blob/9820ff8aed758b2626c068e2fb25629e06540159/pkg/controller/controller.go#L83)
  - [backend pools](https://github.com/Azure/application-gateway-kubernetes-ingress/blob/9820ff8aed758b2626c068e2fb25629e06540159/pkg/controller/controller.go#L90)
  - [health probes](https://github.com/Azure/application-gateway-kubernetes-ingress/blob/9820ff8aed758b2626c068e2fb25629e06540159/pkg/controller/controller.go#L76)

Ingress Controller assumes full ownership and control of the given App Gateway. This makes it impossible to deploy it
to environments with existing pre-configured App Gateways (brownfield deployments). It is also not possible to use AGIC
in hybrid environments where Application Gateway configuration is managed in parts by system administrators as well as
Ingress Controller.

### Proposed Solution
We propose the creation of two new Kubernetes custom resource definitions ([CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)):
  - `AzureIngressManagedLocation` - defines a reference to an App Gateway listener/location. Ingress Controller will assume ownership
  and will mutate configuration for the listener and all related underlying resources: health probes, HTTP settings,
  backend pools etc.
  - `AzureIngressProhibitedLocation` - defines a reference to an App Gateway listener/location, which Ingress Controller is not permitted
    to mutate. This and all related resources and configuration are assumed to be under control by another
    system; Ingress Controller will not make any modifications to these.

These CRDs will be automatically created when the Ingress Controller is deployed to an AKS cluster.

### Proposed CRD schema:
Both `AzureIngressManagedLocation` and `AzureIngressProhibitedLocation` would have the same shape:

##### AzureIngressManagedLocation

```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: azureingressmanagedlocations.appgw.ingress.k8s.io
spec:
  group: appgw.ingress.k8s.io
  version: v1
  names:
    kind: AzureIngressManagedLocation
    plural: azureingressmanagedlocations
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        spec:
          properties:
            ip:
              description: "(required) IP address of the location managed by Ingress Controller; Could be the public or private address attached to the Application Gateway"
              type: string
            host:
              description: "(optional) Hostname of the location"
              type: string
            port:
              description: "(required) Port number of the location"
              type: integer
              minimum: 1
              maximum: 65535
            paths:
              description: "(optional) A list of URL paths, for which the Ingress Controller is allowed to mutate Application Gateway configuration; Must begin with a / and end with /*"
              type: array
              items:
                  type: string
                  pattern: '^\/.*\/\*$'
          required:
            - port
```

##### AzureIngressProhibitedLocation
```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: azureingressprohibitedlocations.appgw.ingress.k8s.io
spec:
  group: appgw.ingress.k8s.io
  version: v1
  names:
    kind: AzureIngressProhibitedLocation
    plural: azureingressprohibitedlocations
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        spec:
          properties:
            ip:
              description: "(required) IP address of the prohibited location; Could be the public or private address attached to the Application Gateway"
              type: string
            host:
              description: "(optional) Hostname of the prohibited location"
              type: string
            port:
              description: "(required) Port number of the prohibited location"
              type: integer
              minimum: 1
              maximum: 65535
            paths:
              description: "(optional) A list of URL paths, for which the Ingress Controller is prohibited from mutating Application Gateway configuration; Must begin with a / and end with /*"
              type: array
              items:
                  type: string
                  pattern: '^\/.*\/\*$'
          required:
            - port
```

### Example
A sample YAML creating new instances of this resource would have the following shape:
```yaml
apiVersion: "appgw.ingress.k8s.io/v1"
kind: AzureIngressManagedLocation
metadata:
  name: ingress-managed-location
spec:
  ip: 23.45.67.89
  host: "www.contoso.com"
  port: 80
  paths:
    - "/foo/*"
    - "/bar/*"

```

The sample `ingress-managed-location` object above will be created by the AKS administrator. It will permit the
Ingress Controller to apply configuration changes only to resources related to (and including) listener/location
for www.contoso.com on ip 23.45.67.89 and port 80 and under path /bar/*


### A Note on Rule Precedence
If a listener/location is referenced in both a `AzureIngressManagedLocation` object as well as a
`AzureIngressProhibitedLocation` object, ingress controller would treat it as `prohibited` to avoid unsafe configuration
mutations. `AzureIngressProhibitedLocation` takes precedence over `AzureIngressManagedLocation`
