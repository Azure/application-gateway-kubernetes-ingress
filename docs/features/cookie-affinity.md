## Enable Cookie based Affinity
As outlined in the [Azure Application Gateway Documentation](https://docs.microsoft.com/en-us/azure/application-gateway/application-gateway-components#http-settings), Application Gateway supports cookie based affinity enabling which it can direct subsequent traffic from a user session to the same server for processing.

### Example
```yaml
apiVersion: networking.k8s.io/v1beta1
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
          serviceName: frontend
          servicePort: 80
```
