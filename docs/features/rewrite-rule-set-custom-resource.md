## Rewrite Rule Set Custom Resource (not released yet)

> Note: This feature is not released yet. Please use [`appgw.ingress.kubernetes.io/rewrite-rule-set`](../annotations.md#rewrite-rule-set) which allows using an existing rewrite rule set on Application Gateway.

Application Gateway allows you to rewrite selected content of requests and responses. With this feature, you can translate URLs, query string parameters as well as modify request and response headers. It also allows you to add conditions to ensure that the URL or the specified headers are rewritten only when certain conditions are met. These conditions are based on the request and response information. Rewrite Rule Set Custom Resource brings this feature to AGIC.

HTTP headers allow a client and server to pass additional information with a request or response. By rewriting these headers, you can accomplish important tasks, such as adding security-related header fields like HSTS/ X-XSS-Protection, removing response header fields that might reveal sensitive information, and removing port information from X-Forwarded-For headers.

With URL rewrite capability, you can:
- Rewrite the host name, path and query string of the request URL
- Choose to rewrite the URL of all requests or only those requests which match one or more of the conditions you set. These conditions are based on the request and response properties (request header, response header and server variables).
- Choose to route the request based on either the original URL or the rewritten URL

## Usage
To use the feature, the customer must define a Custom Resource of the type **AzureApplicationGatewayRewrite** which must have a name in the metadata section. The ingress manifest must refer this Custom Resource via the **`appgw.ingress.kubernetes.io/rewrite-rule-set-custom-resource`** annotation.

## Important points to note:

### metadata & name
- In the metadata section, name of the AzureApplicationGatewayRewrite custom resource should match the custom resource referred in the annotation.

### RuleSequence
- The rule sequence must be unique for every rewrite rule

### Conditions:
You can use rewrite conditions, an optional configuration, to evaluate the content of HTTP(S) requests and responses and perform a rewrite only when one or more conditions are met. 

The following types of variables can be used to define a condition:
- HTTP headers in the request
- HTTP headers in the response
- Application Gateway server variables

**Note:**
- While defining conditions, request headers must be prefixed with `http_req_`, response headers must be prefixed with `http_res_` and list of server variables can be found [here](https://docs.microsoft.com/en-us/azure/application-gateway/rewrite-http-headers-url#server-variables)

### Actions:
You use rewrite actions to specify the URL, request headers or response headers that you want to rewrite and the new value to which you intend to rewrite them to. The value of a URL or a new or existing header can be set to these types of values:

- Text
- Request header
- Response header
- Server Variable
- Combination of the any of the above

**Note:**
- To specify a request header, you need to use the syntax `http_req_headerName`
- To specify a response header, you need to use the syntax `http_resp_headerName`
- To specify a server variable, you need to use the syntax `var_serverVariable`. See the list of supported server variables [here](https://docs.microsoft.com/en-us/azure/application-gateway/rewrite-http-headers-url#server-variables)

#### URL Rewrite Configuration
- URL path: The value to which the path is to be rewritten to.
- URL Query String: The value to which the query string is to be rewritten to.
- Re-evaluate path map: Used to determine whether the URL path map is to be re-evaluated or not. If set to **false**, the original URL path will be used to match the path-pattern in the URL path map. If set to **true**, the URL path map will be re-evaluated to check the match with the rewritten path. 


**Recommended:** More information about Application Gateway's Rewrite feature can be found [here](https://docs.microsoft.com/en-us/azure/application-gateway/rewrite-http-headers-url)

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