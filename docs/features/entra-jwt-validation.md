# Entra JWT validation

> Preview: [JWT validation on Application Gateway](https://learn.microsoft.com/en-us/azure/application-gateway/json-web-token-overview) requires Standard_v2 / WAF_v2 and HTTPS listeners.

AGIC can manage Entra JWT validation configs and attach them to Ingress routing rules.

## Ownership

| Source | Creates / updates JWT config? | Attaches to routing rule? |
| --- | --- | --- |
| Helm `appgw.entraJWT` | Yes (upsert by `name`) | No |
| Ingress `entra-jwt-config-name` | No | Yes |
| Other Azure JWT configs | Untouched | Attach if referenced by name |

Duplicate Helm `name` values fail the chart render (`helm install` / `upgrade`).

## Configure in Helm

```yaml
appgw:
  entraJWT:
    - name: ims-jwt
      tenantId: "<tenant-guid>"
      clientId: "<app-client-id>"
      audiences:
        - "api://my-api"
      unauthorizedAction: Deny
```

## Attach on Ingress

```yaml
metadata:
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/entra-jwt-config-name: "ims-jwt"
```

This is **not** nginx-style `auth-url` / oauth2-proxy. Clients must send `Authorization: Bearer <token>` on each request.
