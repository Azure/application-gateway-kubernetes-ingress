## Troubleshooting: AGIC addon, identity not found

### Illustration
AGIC fails to start due to "Identity not found" even though the User Managed Identity is assigned to the AKS nodes and has the right permissions.

***AGIC Pod Logs***
```
E0813 15:49:23.529297 1 client.go:170] Code="ErrorApplicationGatewayForbidden" Message="Unexpected status code '403' while performing a GET on Application Gateway. You can use 'az role assignment create --role Reader --scope /subscriptions/YOUR_SUBID/resourceGroups/YOUR_RESOURCEGROUP_NAME --assignee YOUR_AGIC_CLIENTID; az role assignment create --role Contributor --scope /subscriptions/YOUR_RESOURCEGROUP_NAME/resourceGroups/YOUR_RESOURCEGROUP_NAME/providers/Microsoft.Network/applicationGateways/YOUR_APPGW_NAME --assignee YOUR_AGIC_CLIENTID' to assign permissions. AGIC Identity needs atleast has 'Contributor' access to Application Gateway 'YOUR_APPGW_NAME' and 'Reader' access to Application Gateway's Resource Group 'YOUR_RESOURCEGROUP_NAME'." InnerError="azure.BearerAuthorizer#WithAuthorization: Failed to refresh the Token for request to https://management.azure.com/subscriptions/YOUR_SUBID/resourceGroups/YOUR_RESOURCEGROUP_NAME/providers/Microsoft.Network/applicationGateways/YOUR_APPGW_NAME?api-version=2019-09-01: StatusCode=403 -- Original Error: adal: Refresh request failed. Status Code = '403'. Response body: failed to get service principal token, error: failed to refresh token, error: adal: Refresh request failed. Status Code = '400'. Response body: {"error":"invalid_request","error_description":"Identity not found"}
```

### Resolve the issue
Team is working on the [fix](https://github.com/Azure/aad-pod-identity/issues/681), currently to resolve the issue above, agic client identity will need to be reassigned to AKS nodes.
Please perform the following commands in your [Azure Cloud Shell](https://shell.azure.com/).
```
aksClusterName="YOUR_AKS_CLUSTER_NAME"
resourceGroup="YOUR_RESOURCEGROUP_NAME"
nodeResourceGroup=$(az aks show -n $aksClusterName -g $resourceGroup --query "nodeResourceGroup" -o tsv)
aksVmssId=$(az vmss list -g $nodeResourceGroup --query "[0].id" -o tsv)
agicIdentity=$(az aks show -n $aksClusterName -g $resourceGroup --query "addonProfiles.ingressApplicationGateway.identity.resourceId" -o tsv)

az vmss identity remove --ids $aksVmssId --identities $agicIdentity
az vmss identity assign --ids $aksVmssId --identities $agicIdentity
```
