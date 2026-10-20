# Allow Custom Error Pages

##### Allow set Custom Error pages in AGIC


### Document Purpose
This document is a proposal to resolve the problem highlighted in [this issue](https://github.com/Azure/application-gateway-kubernetes-ingress/issues/733).

  - Authors: [Renan Netto](https://github.com/renancnetto)
  - Published: February 22th, 2022
  - Status: Open for comments

### Problem Statement
Currently when we set Custom Error Pages using Portal in the Application Gateway, AGIC removes this configuration after a update/deployment. Therefore, there is no way to specify Custom Error Pages permanently.

### Proposed Solution
Using the same pattern mentioned [here](https://azure.github.io/application-gateway-kubernetes-ingress/annotations/), I propose add at least two annotations to allow users to set their Custom Error Pages pointing to publicly blobs.

### Proposed schema
Accept two new annotations in order to configure Custom Error Pages for HTTP Codes 502 (Bad Gateway) and 403 (Forbidden), just like we can do in the Application Gateway trough portal.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: dev-appgateway
  namespace: namespacedev
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/custom-error-502-blob: "https://mypublicblob.blob.core.windows.net/contoso/custom-error-502.html"
    appgw.ingress.kubernetes.io/custom-error-403-blob: "https://mypublicblob.blob.core.windows.net/contoso/custom-error-403.html"
spec:
  rules:
    - host: dev-aks-app.contoso.com
      http:
        paths:
          - path: /articles-fabrikam/*
            pathType: ImplementationSpecific
            backend:
              service:
                name: articles-fabrikam
                port:
                  number: 8080

```
