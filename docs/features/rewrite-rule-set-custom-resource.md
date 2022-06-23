## Rewrite Rule Set Custom Resource
Application Gateway allows you to rewrite selected content of requests and responses. With this feature, you can translate URLs, query string parameters as well as modify request and response headers. It also allows you to add conditions to ensure that the URL or the specified headers are rewritten only when certain conditions are met. These conditions are based on the request and response information. Rewrite Rule Set Custom Resource brings this feature to AGIC.

## Usage
To use the feature, the customer must define a Custom Resource of the type **AzureApplicationGatewayRewrite** which must have a name in the metadata section. The ingress manifest must refer this Custom Resource via the **`appgw.ingress.kubernetes.io/rewrite-rule-set-custom-resource`** annotation.

## Important points to note:
- The rule sequence must be unique for every rewrite rule
- In the metadata section, Name of the AzureApplicationGatewayRewrite custom resource should match the custom resource referred in the annotation.  
- While defining conditions, request headers must be prefixed with `http_req_`, response headers must be prefixed with `http_res_` and list of server variables can be found [here](https://docs.microsoft.com/en-us/azure/application-gateway/rewrite-http-headers-url#server-variables)
- Recommended: More information about AppGw's Rewrite feature can be found [here](https://docs.microsoft.com/en-us/azure/application-gateway/rewrite-http-headers-url)

## Example:
```yaml
apiVersion: appgw.ingress.azure.io/v1beta1
kind: AzureApplicationGatewayRewrite
metadata:
  name: my-rewrite-rule-set-custom-resource
spec:
  rewriteRules:
  - name: rule1
    ruleSequence: 21

    conditions:
    - ignoreCase: false
      negate: false
      variable: http_req_Host
      pattern: example.com

    actions:
      requestHeaderConfigurations:
      - actionType: set
        headerName: incoming-test-header
        headerValue: incoming-test-value
      
      responseHeaderConfigurations:
      - actionType: set
        headerName: outgoing-test-header
        headerValue: outgoing-test-value

      urlConfiguration:
        modifiedPath: "/api/"
        modifiedQueryString: "query=test-value"
        reroute: false

---

apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: my-ingress
  namespace: test-ag
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/rewrite-rule-set-custom-resource: my-rewrite-rule-set
spec:
  rules:
  - http:
      paths:
      - path: /
        pathType: Exact
        backend:
          service:
            name: store-service
            port:
              number: 8080
```