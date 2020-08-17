## Troubleshooting: AGIC fails with aad identity breaking changes

### Illustration
AAD Pod Identity introduced a [breaking change](https://github.com/Azure/aad-pod-identity/tree/v1.6.0#v160-breaking-change) after v1.5.5 regarding CRD fields become case sensitive.
When using user assigned identity with AGIC version < v1.2.0-rc2, in case to apply AAD Pod Identity version >= 1.6.0 or from AAD master branch such as https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment-rbac.yaml, an error as shown below will be raised due to the breaking change, the error is caused by AAD Pod Identiy fields are mismatched to what AGIC uses.

***AGIC Pod Logs***
```
E0428 16:57:55.669130       1 client.go:132] Possible reasons: AKS Service Principal requires 'Managed Identity Operator' access on Controller Identity; 'identityResourceID' and/or 'identityClientID' are incorrect in the Helm config; AGIC Identity requires 'Contributor' access on Application Gateway and 'Reader' access on Application Gateway's Resource Group;
E0428 16:57:55.669160       1 client.go:145] Unexpected ARM status code on GET existing App Gateway config: 403
E0428 16:57:55.669167       1 client.go:148] Failed fetching config for App Gateway instance. Will retry in 10s. Error: azure.BearerAuthorizer#WithAuthorization: Failed to refresh the Token for request to https://management.azure.com/subscriptions/4c4aee1a-cfd4-4e7a-abe3-*******/resourceGroups/RG-NAME-DEV/providers/Microsoft.Network/applicationGateways/AG-NAME-DEV?api-version=2019-09-01: StatusCode=403 -- Original Error: adal: Refresh request failed. Status Code = '403'. Response body: getting assigned identities for pod default/agile-opossum-ingress-azure-579cbb6b89-sldr5 in CREATED state failed after 16 attempts, retry duration [5]s. Error: <nil>
```

***MIC Pod Logs***
```
E0427 00:13:26.222815       1 mic.go:899] Ignoring azure identity default/agic-azid-ingress-azure, error: Invalid resource id: "", must match /subscriptions/<subid>/resourcegroups/<resourcegroup>/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<name>
```

### Analysis of the issue
#### AAD Breaking Change details
For `AzureIdentity` and `AzureIdentityBinding` created using AAD Pod Identity v1.6.0+, the following fields are changed
 ***AzureIdentity***

| < 1.6.0          | >= 1.6.0         |
|------------------|------------------|
| `ClientID`       | `clientID`       |
| `ClientPassword` | `clientPassword` |
| `ResourceID`     | `resourceID`     |
| `TenantID`       | `tenantID`       |

***AzureIdentityBinding***

| < 1.6.0         | >= 1.6.0        |
|-----------------|-----------------|
| `AzureIdentity` | `azureIdentity` |
| `Selector`      | `selector`      |

#### AGIC fix to adapt to the break change
Updated AGIC helm templates to use the right fields regarding AAD Pod Identity, [PR](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/825/files) for reference.


### Resolve the issue

#### Upgrade AGIC
AGIC version at least [v1.2.0-rc2](https://github.com/Azure/application-gateway-kubernetes-ingress/blob/master/CHANGELOG/CHANGELOG-1.2.md#v120-rc2) or release [v1.2.0](https://github.com/Azure/application-gateway-kubernetes-ingress/releases/tag/1.2.0) will be required.

```
# https://github.com/Azure/application-gateway-kubernetes-ingress/blob/master/docs/how-tos/helm-upgrade.md
# --reuse-values   when upgrading, reuse the last release's values and merge in any overrides from the command line via --set and -f. If '--reset-values' is specified, this is ignored
helm repo update
helm upgrade \
  <release-name> \
  application-gateway-kubernetes-ingress/ingress-azure \
  --reuse-values \
  --version 1.2.0-rc2
```

#### Install the right version of AAD Pod Idenity
if old version of AGIC being used, instead of upgrading AGIC as above, reinstall AAD Pod Identity with the right version, e.g. v1.5.5

- *RBAC enabled* AKS cluster

```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/v1.5.5/deploy/infra/deployment-rbac.yaml
```

- *RBAC disabled* AKS cluster

```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/v1.5.5/deploy/infra/deployment.yaml
```
