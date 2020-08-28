## Troubleshooting: AGIC v1.2.0-rc1 and below fails with a breaking change introduced in AAD Pod Identity v1.6 

### Overview
If you're using AGIC with version < v1.2.0-rc2 and AAD Pod Identity with version >= v1.6.0, an error as shown below will be raised due to a breaking change. AAD Pod Identity introduced a [breaking change](https://github.com/Azure/aad-pod-identity/tree/v1.6.0#v160-breaking-change) after v1.5.5 due to CRD fields being case sensitive. The error is caused by AAD Pod Identity fields not matching what AGIC uses; more details of the mismatch under [analysis of the issue](#analysis-of-the-issue). AAD Pod Identity v1.5 and lower have known issues with AKS' most recent base images, and therefore AKS has asked customers to upgrade to AAD Pod Identity v1.6 or higher. 

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
#### AAD breaking change details
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

***NOTE*** AKS recommends to using AAD Pod Identity with version >= 1.6.0

#### AGIC fix to adapt to the breaking change
Updated AGIC Helm templates to use the right fields regarding AAD Pod Identity, [PR](https://github.com/Azure/application-gateway-kubernetes-ingress/pull/825/files) for reference.


### Resolving the issue
It's recommended you upgrade your AGIC to release 1.2.0 and then apply AAD Pod Identity version >= 1.6.0
#### Upgrade AGIC to 1.2.0
AGIC version [v1.2.0](https://github.com/Azure/application-gateway-kubernetes-ingress/releases/tag/1.2.0) will be required.

```
# https://github.com/Azure/application-gateway-kubernetes-ingress/blob/master/docs/how-tos/helm-upgrade.md
# --reuse-values   when upgrading, reuse the last release's values and merge in any overrides from the command line via --set and -f. If '--reset-values' is specified, this is ignored

helm repo update

# check the latest relese version of AGIC
helm search repo -l application-gateway-kubernetes-ingress

# install release 1.2.0
helm upgrade \
  <release-name> \
  application-gateway-kubernetes-ingress/ingress-azure
  --version 1.2.0
  --reuse-values
```
***Note:*** If you're upgrading from v1.0.0 or below, you'll have to delete AGIC and then reinstall with v1.2.0. 


#### Install the right version of AAD Pod Identity
AKS recommends upgrading the Azure Active Directory Pod Identity version on your Azure Kubernetes Service Clusters to v1.6. AAD pod identity v1.5 or lower have a known issue with AKS' most recent base images. 

To install AAD Pod Identity with version v1.6.0:
- *RBAC enabled* AKS cluster

```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/v1.6.0/deploy/infra/deployment-rbac.yaml
```

- *RBAC disabled* AKS cluster

```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/aad-pod-identity/v1.6.0/deploy/infra/deployment.yaml
```
