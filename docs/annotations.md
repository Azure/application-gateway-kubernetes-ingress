# Table of Contents
- [Introduction](#introduction)
- [Backend path prefix](#backend-path-prefix)

## Introduction
Kubernetes Ingress specification allows for annotations. We use annotations to expose Application Gateway specific features that can't be exposed using the ingress specification. It is important to note that annotations defined on an ingress resource are applied to all HTTP setting, backend pools and listeners defined within a given ingress resource.

## Bakend path prefix
This annotation allows the backend path specified in an ingress resource to be re-written with prefix specified in this annotation. This allows users to expose services whose endpoints are different than endpoint names used to expose a service in an ingress resource.

### Usage
```yaml
appgw.ingress.kubernetes.io/backend-path-prefix: <path prefix>
```

### Example
```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: go-server-ingress-bkprefix
  namespace: test-ag
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/backend-path-prefix: "/test/"
spec:
  rules:
  - http:
      paths:
      - path: /hello/
        backend:
          serviceName: go-server-service
          servicePort: 80
```
In the example above we have defined an ingress resource named `go-server-ingress-bkprefix` with an annotation `appgw.ingress.kubernetes.io/backend-path-prefix: "/test/"` . The annotation tells application gateway to create an HTTP setting which will have a path prefix override for the path `/hello` to `/test/`. 

***NOTE:*** In the above example we have only one rule defined. However, the annotations is applicable to the entire ingress resource so if a user had defined multiple rules the backend path prefix would be setup for each of the paths sepcified. Thus, if a user wants different rules with different path prefixes (even for the same service) they would need to define different ingress resources. 
