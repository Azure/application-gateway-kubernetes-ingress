## Enable Cookie based Affinity

> **_NOTE:_** [Application Gateway for Containers](https://aka.ms/agc) has been released, which introduces numerous performance, resilience, and feature changes. Please consider leveraging Application Gateway for Containers for your next deployment. Details on cookie based affinity for Application Gateway for Containers [may be found here](https://learn.microsoft.com/azure/application-gateway/for-containers/session-affinity?tabs=session-affinity-gateway-api).

As outlined in the [Azure Application Gateway Documentation](https://docs.microsoft.com/en-us/azure/application-gateway/application-gateway-components#http-settings), Application Gateway supports cookie based affinity enabling which it can direct subsequent traffic from a user session to the same server for processing.

### Example

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: guestbook
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/cookie-based-affinity: "true"
spec:
  rules:
  - http:
      paths:
      - backend:
          service:
            name: frontend
            port:
              number: 80
```
