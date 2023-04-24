# Custom Ingress Class
> **Minimum version:** 1.3.0

Custom ingress class allows you to customize the ingress class selector that AGIC will use when filtering the ingress manifests. AGIC uses `azure/application-gateway` as default ingress class. This  will allow you to target multiple AGICs on a single namespace as each AGIC can now use it's own ingress class.

For instance, AGIC with ingress class `agic-public` can serves public traffic, and AGIC wit `agic-private` can serve "internal" traffic.

To use a custom ingress class,

1. Install AGIC by providing a value for `kubernetes.ingressClass` in helm config.
    ```bash
    helm install ./helm/ingress-azure \
        --name ingress-azure \
        -f helm-config.yaml
        --set kubernetes.ingressClass arbitrary-class
    ```

2. Then, within the spec object, specify `ingressClassName` with the same value to provided to AGIC.
    ```yaml
    kind: Ingress
    metadata:
    name: go-server-ingress-affinity
    namespace: test-ag
    spec:
      ingressClassName: arbitrary-class
      rules:
      - http:
        paths:
          - path: /hello/
            backend:
              service:
                name: store-service
                port:
                  number: 80
    ```

## Reference
* [Proposal Document](../../proposals\multiple-gateways-single-cluster.md)