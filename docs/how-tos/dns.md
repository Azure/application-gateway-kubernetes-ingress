# Automate DNS updates

When a hostname is specified in the Kubernetes Ingress resource's rules, it can be used to automatically create DNS records for the given domain and App Gateway's IP address.
To achieve this the [ExternalDNS](https://github.com/kubernetes-sigs/external-dns) Kubernetes app is required. ExternalDNS in installable via a [Helm chart](https://github.com/kubernetes-incubator/external-dns). The [following document](https://github.com/kubernetes-incubator/external-dns/blob/master/docs/tutorials/azure.md) provides a tutorial on setting up ExternalDNS with an Azure DNS.

Below is a sample Ingress resource, annotated with
`kubernetes.io/ingress.class: azure/application-gateway`, which configures `aplpha.contoso.com`.

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: websocket-ingress
  namespace: alpha
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
spec:
  rules:
    - host: alpha.contoso.com
      http:
        paths:
        - path: /
          backend:
            service:
              name: contoso-service
              port:
                number: 80
          pathType: Exact
```

Application Gateway Ingress Controller (AGIC) automatically recognizes the public IP address
assigned to the Application Gateway it is associated with, and sets this IP (`1.2.3.4`)
on the Ingress resource as shown below:

```bash
$ kubectl get ingress -A
NAMESPACE             NAME                HOSTS                 ADDRESS   PORTS   AGE
alpha                 alpha-ingress       alpha.contoso.com     1.2.3.4   80      8m55s
beta                  beta-ingress        beta.contoso.com      1.2.3.4   80      8m54s

```

Once the Ingresses contain both host and adrress, ExternalDNS will provision these to the
DNS system it has been associated with and authorized for.
