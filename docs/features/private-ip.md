# Using Private IP for internal routing

## Pre-requisites
* Application Gateway with a [Private IP configuration](https://docs.microsoft.com/en-us/azure/application-gateway/configure-application-gateway-with-private-frontend-ip)

This feature allows to expose the ingress endpoint within the `Virtual Network`.
To configure the controller to use Private IP for routing, modify the `helm` config by adding `usePrivateIP: true`

## Example
```yaml
appgw:
    subscriptionId: <subscriptionId>
    resourceGroup: <resourceGroupName>
    name: <applicationGatewayName>
    usePrivateIP: true

armAuth:
     type: aadPodIdentity
     identityResourceID: <identityResourceId>
     identityClientID:  <identityClientId>

```

This will make the ingress controller filter the ipconfigurations for a Private IP when configuring the frontend listeners on the Application Gateway.
Controller will panic and crash if `usePrivateIP: true` and no Private IP is assigned.

Notes:
1) Application Gateway v2 SKU manadates a Public IP. For meeting compliance requirement where the Application Gateway should be completely private, Attach a [`Network Security Group`](https://docs.microsoft.com/en-us/azure/virtual-network/security-overview) to the Application Gateway's subnet to restrict traffic.
2) To expose the Ingress both on public and private, current recommedation is to deploy two Application Gateways with respective Ingress Controllers, one with Public IP and one with Private IP.
