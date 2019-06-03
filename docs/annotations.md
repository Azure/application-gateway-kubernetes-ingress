## Introduction
Kubernetes Ingress specification allows for annotations. We use annotations to expose Application Gateway specific features that can't be exposed using the ingress specification. It is important to note that annotations defined on an ingress resource are applied to all HTTP setting, backend pools and listeners defined within a given ingress resource.

### List of supported annotations
| Annotation Key | Value Type | Default Value |
| -- | -- | -- |
| [appgw.ingress.kubernetes.io/backend-path-prefix](#backend-path-prefix) | `string` | `nil` |
| [appgw.ingress.kubernetes.io/ssl-redirect](#ssl-redirect) | `bool` | `false` |  |
| [appgw.ingress.kubernetes.io/connection-draining](#connection-draining) | `bool` | `false` |
| [appgw.ingress.kubernetes.io/connection-draining-timeout](#connection-draining) | `int32` (seconds) | `30` |
| [appgw.ingress.kubernetes.io/cookie-based-affinity](#cookie-based-affinity) | `bool` | `false` |
| [appgw.ingress.kubernetes.io/request-timeout](#request-timeout) | `int32` (seconds) | `30` |

## Backend Path Prefix
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

## SSL Redirect
Application Gateway [can be configured](https://docs.microsoft.com/en-us/azure/application-gateway/application-gateway-redirect-overview)
to automatically redirect HTTP URLs to their HTTPS counterparts. When this
annotation is present and TLS is properly configured, Kubernetes Ingress
controller will create a [routing rule with a redirection configuration](https://docs.microsoft.com/en-us/azure/application-gateway/redirect-http-to-https-portal#add-a-routing-rule-with-a-redirection-configuration)
and apply the changes to your App Gateway. The redirect created will be HTTP `301 Moved Permanently`.

### Usage
```yaml
appgw.ingress.kubernetes.io/ssl-redirect: "true"
```

### Example
```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: go-server-ingress-redirect
  namespace: test-tag
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/ssl-redirect: "true"
spec:
  tls:
   - hosts:
     - www.contoso.com
     secretName: testsecret-tls
  rules:
  - host: www.contoso.com
    http:
      paths:
      - backend:
          serviceName: websocket-repeater
          servicePort: 80
```

# Connection Draining
`connection-draining`: This annotation allows to specify whether to enable connection draining.
`connection-draining-timeout`: This annotation allows to specify a timeout after which Application Gateway will terminate the requests to the draining backend endpoint.

### Usage
```yaml
appgw.ingress.kubernetes.io/connection-draining: "true"
appgw.ingress.kubernetes.io/connection-draining-timeout: 60
```

### Example
```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: go-server-ingress-drain
  namespace: test-ag
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/connection-draining: "true"
    appgw.ingress.kubernetes.io/connection-draining-timeout: 60
spec:
  rules:
  - http:
      paths:
      - path: /hello/
        backend:
          serviceName: go-server-service
          servicePort: 80
```

# Cookie Based Affinity
This annotation allows to specify whether to enable cookie based affinity.

### Usage
```yaml
appgw.ingress.kubernetes.io/cookie-based-affinity: "true"
```

### Example
```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: go-server-ingress-affinity
  namespace: test-ag
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/cookie-based-affinity: "true"
spec:
  rules:
  - http:
      paths:
      - path: /hello/
        backend:
          serviceName: go-server-service
          servicePort: 80
```

# Request Timeout
This annotation allows to specify the request timeout in seconds after which Application Gateway will fail the request if response is not received.

### Usage
```yaml
appgw.ingress.kubernetes.io/request-timeout: 20
```

### Example
```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: go-server-ingress-timeout
  namespace: test-ag
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/request-timeout: 20
spec:
  rules:
  - http:
      paths:
      - path: /hello/
        backend:
          serviceName: go-server-service
          servicePort: 80
```