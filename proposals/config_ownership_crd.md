# CRD for configuration ownership in brownfield deployments

##### Deploying Ingress Controller to hybrid environments with partial Application Gateway configuration ownership


### Document Purpose
This document is a proposal for the creation of a new [CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/) with the goal of
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
We propose the creation of a new Kubernetes custom resource definitions ([CRD](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/)):
  - `AzureIngressProhibitedTarget` - defines a reference to an App Gateway listener/target, which Ingress Controller is not permitted
    to mutate. This and all related resources and configuration are assumed to be under control by another
    system; Ingress Controller will not make any modifications to these.

This CRD will be automatically created when the Ingress Controller is deployed to an AKS cluster.

### Proposed CRD schema:
`AzureIngressProhibitedTarget` would have the following shape:

```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: azureingressprohibitedtargets.appgw.ingress.k8s.io
spec:
  group: appgw.ingress.k8s.io
  version: v1
  names:
    kind: AzureIngressProhibitedTarget
    plural: azureingressprohibitedtargets
  scope: Namespaced
  validation:
    openAPIV3Schema:
      properties:
        spec:
          properties:
            hostname:
              description: "Hostname of the prohibited target"
              type: string
            paths:
              description: "A list of URL paths, for which the Ingress Controller is prohibited from mutating Application Gateway configuration; Must begin with a / and end with /*"
              type: array
              items:
                  type: string
                  pattern: '^\/.*\/\*$'
```

### Example
A sample YAML creating new instances of this resource would have the following shape:
```yaml
apiVersion: "appgw.ingress.k8s.io/v1"
kind: AzureIngressProhibitedTarget
metadata:
  name: ingress-prohibited-target
spec:
  hostname: "www.contoso.com"
  paths:
    - "/foo/*"
    - "/bar/*"
```

The sample `ingress-prohibited-target` object above will be created by the AKS administrator. It will prohibit the
Ingress Controller from applying configuration changes to resources related to (and including) listener/target
for www.contoso.com under path `/bar/*`.
