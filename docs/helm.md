# Helm Configuration Options

## Available options

| Field | Default | Description |
| - | - | - |
| `verbosityLevel`| 3 | Sets the verbosity level of the AGIC logging infrastructure. See [Logging Levels](troubleshooting.md#logging-levels) for possible values. |
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
# This file contains the essential configs for the ingress controller helm chart

# Verbosity level of the App Gateway Ingress Controller
verbosityLevel: 3

################################################################################
# Specify which application gateway the ingress controller will manage
#
appgw:
    applicationGatewayID: <application-gateway-resource-id>
    # or
    # subscriptionId: <subscriptionId>
    # resourceGroup: <resourceGroupName>
    # name: <applicationGatewayName>

    # If you want AGIC to deploy a new gateway, You must provide subnet information like subnetPrefix/subnetName or subnetId.
    # subnetPrefix: <subnetPrefix>
    # subnetName: <subnetName>
    # Example -
    # subnetPrefix: 10.1.0.0/16
    # subnetName: appgw-subnet
    # or
    # subnetID: <subnet-resource-id>

    # Setting appgw.shared to "true" will create an AzureIngressProhibitedTarget CRD.
    # This prohibits AGIC from applying config for any host/path.
    # Use "kubectl get AzureIngressProhibitedTargets" to view and change this.
    shared: false

################################################################################
# Specify which kubernetes namespace the ingress controller will watch
# Default value is "default"
# Leaving this variable out or setting it to blank or empty string would
# result in Ingress Controller observing all acessible namespaces.
#
# kubernetes:
#   watchNamespace: <namespace>

################################################################################
# Specify the authentication with Azure Resource Manager
#
# Two authentication methods are available:
# - Option 1: AAD-Pod-Identity (https://github.com/Azure/aad-pod-identity)
armAuth:
    type: aadPodIdentity
    identityResourceID: <identityResourceId>
    identityClientID:  <identityClientId>

## Alternatively you can use Service Principal credentials
# armAuth:
#    type: servicePrincipal
#    secretJSON: <<Generate this value with: "az ad sp create-for-rbac --subscription <subscription-uuid> --sdk-auth | base64 -w0" >>

################################################################################
# Specify if the cluster is RBAC enabled or not
rbac:
    enabled: false # true/false
```