# Helm Configuration Options

## Available options

| Field | Default | Description |
| - | - | - |
| `verbosityLevel`| 3 | Sets the verbosity level of the AGIC logging infrastructure. See [Logging Levels](troubleshooting.md#logging-levels) for possible values. |
| `reconcilePeriodSeconds` | | Enable periodic reconciliation to checks if the latest gateway configuration is different from what it cached. Range: 30 - 300 seconds. Disabled by default. |
| `appgw.applicationGatewayId` | | Resource Id of the Application Gateway. Example: `applicationgatewayd0f0` |
| `appgw.subscriptionId` | Default is agent node pool's subscriptionId derived from CloudProvider config  | The Azure Subscription ID in which App Gateway resides. Example: `a123b234-a3b4-557d-b2df-a0bc12de1234` |
| `appgw.resourceGroup` | Default is agent node pool's resource group derived from CloudProvider config | Name of the Azure Resource Group in which App Gateway was created. Example: `app-gw-resource-group` |
| `appgw.name` | | Name of the Application Gateway. Example: `applicationgatewayd0f0` |
| `appgw.subnetId` | | Resource Id of an existing Subnet used to deploy the Application Gateway |
| `appgw.subnetPrefix` | | Subnet Prefix of the Subnet. A new subnet will be created using appgw.subnetPrefix and appgw.subnetName if no subnet is found with matching subnet prefix or subnet name. Example: 10.1.0.0/16 |
| `appgw.subnetName` | {{ appgw.name }}-subnet | Name of the subnet. A new subnet will be created using appgw.subnetPrefix and appgw.subnetName if no subnet is found with matching subnet prefix or subnet name. Example: appgw-subnet |
| `appgw.shared` | false | This boolean flag should be defaulted to `false`. Set to `true` should you need a [Shared App Gateway](setup/install-existing.md#multi-cluster--shared-app-gateway). |
| `kubernetes.watchNamespace` | Watches all if empty | Specify the name space, which AGIC should watch. This could be a single string value, or a comma-separated list of namespaces. |
| `kubernetes.rbac` | false | Specify true if kubernetes cluster is rbac enabled |
| `armAuth.type` | | could be `aadPodIdentity` or `servicePrincipal` |
| `armAuth.identityResourceID` | | Resource ID of the Azure Managed Identity |
| `armAuth.identityClientId` | | The Client ID of the Identity. See below for more information on Identity |
| `armAuth.secretJSON` | | Only needed when Service Principal Secret type is chosen (when `armAuth.type` has been set to `servicePrincipal`) |

## Example

```yaml
appgw:
    applicationGatewayID: <application-gateway-resource-id>

armAuth:
    type: aadPodIdentity
    identityResourceID: <identityResourceId>
    identityClientID:  <identityClientId>

rbac:
    enabled: false
```