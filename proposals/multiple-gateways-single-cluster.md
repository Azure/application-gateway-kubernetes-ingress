# Multiple Gateways Single Cluster

##### Deploying multiple ingress controller instances to the same cluster


### Document Purpose
This document is a proposal to resolve the problem highlighted in [this issue](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/732).

  - Authors: [Garvin Casimir](https://github.com/garvinmsft)
  - Published: February 25th, 2020
  - Status: Open for comments

### Problem Statement
Currently the only way to partition AGIC controllers is via namespacing. Therefore, there is no way to delegate 2 ingress resources in the same namespace to 2 different controllers.

### Proposed Solution
Using the same pattern mentioned [here](https://kubernetes.github.io/ingress-nginx/user-guide/multiple-ingress/#multiple-ingress-nginx-controllers), I propose extending the ingress class annotation to allow users to partition by arbitrary keys.

### Option 1: Allow any arbitrary string
Accept entire ingress class name as a parameter in the helm chart. The danger with this approach is that it leaves the door open for someone to inadvertently enter an ingress class name already being used by a different type of ingress controller.

```bash
helm install ./helm/ingress-azure \
     --name ingress-azure \
     --ingress-class  arbitrary-class
```

```yaml
kind: Ingress
metadata:
  name: go-server-ingress-affinity
  namespace: test-ag
  annotations:
    kubernetes.io/ingress.class: arbitrary-class
spec:
  rules:
  - http:
      paths:
      - path: /hello/
        backend:
          service:
            name: go-server-service
            port:
              number: 80
```

### Option 2: Enforce a prefix unique to AGIC
This option is similar to option 1 except the helm parameter is used as a suffix.

```bash
helm install ./helm/ingress-azure \
     --name ingress-azure \
     --ingress-suffix arbitrary-class
```

```yaml
kind: Ingress
metadata:
  name: go-server-ingress-affinity
  namespace: test-ag
  annotations:
    kubernetes.io/ingress.class: agic-arbitrary-class
spec:
  rules:
  - http:
      paths:
      - path: /hello/
        backend:
          service:
            name: go-server-service
            port:
              number: 80
```
