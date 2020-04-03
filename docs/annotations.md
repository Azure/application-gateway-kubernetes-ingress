# Annotations

## Introductions

The Kubernetes Ingress resource can be annotated with arbitrary key/value pairs. AGIC relies on annotations to program Application Gateway features, which are not configurable via the Ingress YAML. Ingress annotations are applied to all HTTP setting, backend pools and listeners derived from an ingress resource.

## List of supported annotations

For an Ingress resource to be observed by AGIC it **must be annotated** with `kubernetes.io/ingress.class: azure/application-gateway`. Only then AGIC will work with the Ingress resource in question.

| Annotation Key | Value Type | Default Value | Allowed Values
| -- | -- | -- | -- |
| [appgw.ingress.kubernetes.io/backend-path-prefix](#backend-path-prefix) | `string` | `nil` | |
| [appgw.ingress.kubernetes.io/backend-hostname](#backend-hostname) | `string` | `nil` | |
| [appgw.ingress.kubernetes.io/backend-protocol](#backend-protocol) | `string` | `http` | `http`, `https` |
| [appgw.ingress.kubernetes.io/ssl-redirect](#ssl-redirect) | `bool` | `false` | |
| [appgw.ingress.kubernetes.io/appgw-ssl-certificate](#appgw-ssl-certificate) | `string` | `nil` | |
| [appgw.ingress.kubernetes.io/connection-draining](#connection-draining) | `bool` | `false` | |
| [appgw.ingress.kubernetes.io/connection-draining-timeout](#connection-draining) | `int32` (seconds) | `30` | |
| [appgw.ingress.kubernetes.io/cookie-based-affinity](#cookie-based-affinity) | `bool` | `false` | |
| [appgw.ingress.kubernetes.io/request-timeout](#request-timeout) | `int32` (seconds) | `30` | |
| [appgw.ingress.kubernetes.io/use-private-ip](#use-private-ip) | `bool` | `false` | |
| [appgw.ingress.kubernetes.io/waf-policy-for-path](#azure-waf-policy-for-path) | `string` |   |   |

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

***NOTE:*** In the above example we have only one rule defined. However, the annotations is applicable to the entire ingress resource so if a user had defined multiple rules the backend path prefix would be setup for each of the paths specified. Thus, if a user wants different rules with different path prefixes (even for the same service) they would need to define different ingress resources.

## Backend Hostname

This annotations allows us to specify the host name that Application Gateway should use while talking to the Pods.

### Usage
```yaml
appgw.ingress.kubernetes.io/backend-hostname: "internal.example.com"
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
    appgw.ingress.kubernetes.io/backend-hostname: "internal.example.com"
spec:
  rules:
  - http:
      paths:
      - path: /hello/
        backend:
          serviceName: go-server-service
          servicePort: 80
```

## Backend Protocol

This annotation allows us to specify the protocol that Application Gateway should use while talking to the Pods. Supported Protocols: `http`, `https`

> **Note**
1) While self-signed certificates are supported on Application Gateway, currently, AGIC only support `https` when Pods are using certificate signed by a well-known CA.
2) Make sure to not use port 80 with HTTPS and port 443 with HTTP on the Pods.

### Usage
```yaml
appgw.ingress.kubernetes.io/backend-protocol: "https"
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
    appgw.ingress.kubernetes.io/backend-protocol: "https"
spec:
  rules:
  - http:
      paths:
      - path: /hello/
        backend:
          serviceName: go-server-service
          servicePort: 443
```

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
  namespace: test-ag
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

## AppGw SSL Certificate

The SSL certificate [can be configured to Application Gateway](https://docs.microsoft.com/en-us/cli/azure/network/application-gateway/ssl-cert?view=azure-cli-latest#az-network-application-gateway-ssl-cert-create) either from a local PFX cerficate file or a reference to a Azure Key Vault unversioned secret Id.
When the annotation is present with a certificate name and the certificate is pre-installed in Application Gateway, Kubernetes Ingress controller will create a routing rule with a HTTPS listener and apply the changes to your App Gateway.
`appgw-ssl-certificate` annotation can also be used together with `ssl-redirect` annotation in case of SSL redirect.

> **Note**
* Annotation "appgw-ssl-certificate" will be ignored when TLS Spec is defined in ingress at the same time.
* If a user wants different certs with different hosts, they would need to define different ingress resources.

### Use Azure CLI to install certificate to Application Gateway
* Configure from a local PFX certificate file
```bash
az network application-gateway ssl-cert create -g $resgp  --gateway-name $appgwName -n mysslcert --cert-file \path\to\cert\file --cert-password Abc123
```

* Configure from a reference to a Key Vault unversioned secret id
```bash
az keyvault certificate create --vault-name $vaultName -n cert1 -p "$(az keyvault certificate get-default-policy)"
versionedSecretId=$(az keyvault certificate show -n cert --vault-name $vaultName --query "sid" -o tsv)
unversionedSecretId=$(echo $versionedSecretId | cut -d'/' -f-5) # remove the version from the url
az network application-gateway ssl-cert create -n mysslcert --gateway-name $appgwName --resource-group $resgp --key-vault-secret-id $unversionedSecretId
```

### Usage

```yaml
appgw.ingress.kubernetes.io/appgw-ssl-certificate: "name-of-appgw-installed-certificate"
```

### Example

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: go-server-ingress-certificate
  namespace: test-ag
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/appgw-ssl-certificate: "name-of-appgw-installed-certificate"
spec:
  rules:
  - host: www.contoso.com
    http:
      paths:
      - backend:
          serviceName: websocket-repeater
          servicePort: 80
```

## Connection Draining

`connection-draining`: This annotation allows to specify whether to enable connection draining.
`connection-draining-timeout`: This annotation allows to specify a timeout after which Application Gateway will terminate the requests to the draining backend endpoint.

### Usage

```yaml
appgw.ingress.kubernetes.io/connection-draining: "true"
appgw.ingress.kubernetes.io/connection-draining-timeout: "60"
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
    appgw.ingress.kubernetes.io/connection-draining-timeout: "60"
spec:
  rules:
  - http:
      paths:
      - path: /hello/
        backend:
          serviceName: go-server-service
          servicePort: 80
```

## Cookie Based Affinity

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

## Request Timeout

This annotation allows to specify the request timeout in seconds after which Application Gateway will fail the request if response is not received.

### Usage

```yaml
appgw.ingress.kubernetes.io/request-timeout: "20"
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
    appgw.ingress.kubernetes.io/request-timeout: "20"
spec:
  rules:
  - http:
      paths:
      - path: /hello/
        backend:
          serviceName: go-server-service
          servicePort: 80
```

## Use Private IP

This annotation allows us to specify whether to expose this endpoint on Private IP of Application Gateway.

> **Note**
1) App Gateway doesn't support multiple IPs on the same port (example: 80/443). Ingress with annotation `appgw.ingress.kubernetes.io/use-private-ip: "false"` and another with `appgw.ingress.kubernetes.io/use-private-ip: "true"` on `HTTP` will cause AGIC to fail in updating the App Gateway.
2) For App Gateway that doesn't have a private IP, Ingresses with `appgw.ingress.kubernetes.io/use-private-ip: "true"` will be ignored. This will reflected in the controller logs and ingress events for those ingresses with `NoPrivateIP` warning.


### Usage
```yaml
appgw.ingress.kubernetes.io/use-private-ip: "true"
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
    appgw.ingress.kubernetes.io/use-private-ip: "true"
spec:
  rules:
  - http:
      paths:
      - path: /hello/
        backend:
          serviceName: go-server-service
          servicePort: 80
```


## Attach firewall policy to a host and path
This annotation allows you to attach an already created WAF policy to the list paths for a host within a Kubernetes
Ingress resource being annotated.

The WAF policy must be created in advance. Example of using [Azure Portal](https://portal.azure.com/) to create a policy:
![Creating a WAF policy](./images/waf-policy.png)

Once the policy is created, copy the URI of the policy from the address bar of Azure Portal:
![Creating a WAF policy](./images/waf-policy-1.png)

The URI would have the following format:
```bash
/subscriptions/<YOUR-SUBSCRIPTION>/resourceGroups/<YOUR-RESOURCE-GROUP>/providers/Microsoft.Network/applicationGatewayWebApplicationFirewallPolicies/<YOUR-POLICY-NAME>
```

### Usage

```yaml
appgw.ingress.kubernetes.io/waf-policy-for-path: "/subscriptions/abcd/resourceGroups/rg/providers/Microsoft.Network/applicationGatewayWebApplicationFirewallPolicies/adserver"
```

### Example
The example below will apply the WAF policy 
```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: ad-server-ingress
  namespace: commerce
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/waf-policy-for-path: "/subscriptions/abcd/resourceGroups/rg/providers/Microsoft.Network/applicationGatewayWebApplicationFirewallPolicies/adserver"
spec:
  rules:
  - http:
      paths:
      - path: /ad-server
        backend:
          serviceName: ad-server
          servicePort: 80
          
      - path: /auth
        backend:
          serviceName: auth-server
          servicePort: 80
```
Note that the WAF policy will be applied to both `/ad-server` and `/auth` URLs.