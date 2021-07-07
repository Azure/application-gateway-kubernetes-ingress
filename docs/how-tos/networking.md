# How to setup networking between Application Gateway and AKS

When you are using Application Gateway with AKS for L7, you need to make sure that you have setup network connectivity correctly between the gateway and the cluster. Otherwise, you might receive 502s when reaching your site.

There are two major things to consider when setting up network connectivity between Application Gateway and AKS
1. Virtual Network Configuration
    * When AKS and Application Gateway in the [same virtual network](#deployed-in-same-vnet)
    * When AKS and Application Gateway in [different virtual networks](#deployed-in-different-vnets)
1. Network Plugin used with AKS
    * [Kubenet](#with-kubenet)
    * [Azure(advanced) CNI](#with-azure-cni)

## Virtual Network Configuration
### Deployed in same virtual network
If you have deployed AKS and Application Gateway in the same virtual network with `Azure CNI` for network plugin, then you don't have to do any changes and you are good to go. Application Gateway instances should be able to reach the PODs.

If you are using `kubenet` network plugin, then jump to [Kubenet](#with-kubenet) to setup the route table.

### Deployed in different vnets
AKS can be deployed in different virtual network from Application Gateway's virtual network, however, the two virtual networks must be [peered](https://docs.microsoft.com/en-us/azure/virtual-network/virtual-network-peering-overview) together. When you create a virtual network peering between two virtual networks, a route is added by Azure for each address range within the address space of each virtual network a peering is created for.

```bash
aksClusterName="<aksClusterName>"
aksResourceGroup="<aksResourceGroup>"
appGatewayName="<appGatewayName>"
appGatewayResourceGroup="<appGatewayResourceGroup>"

# get aks vnet information
nodeResourceGroup=$(az aks show -n $aksClusterName -g $aksResourceGroup -o tsv --query "nodeResourceGroup")
aksVnetName=$(az network vnet list -g $nodeResourceGroup -o tsv --query "[0].name")
aksVnetId=$(az network vnet show -n $aksVnetName -g $nodeResourceGroup -o tsv --query "id")

# get gateway vnet information
appGatewaySubnetId=$(az network application-gateway show -n $appGatewayName -g $appGatewayResourceGroup -o tsv --query "gatewayIpConfigurations[0].subnet.id")
appGatewayVnetName=$(az network vnet show --ids $appGatewaySubnetId -o tsv --query "name")
appGatewayVnetId=$(az network vnet show --ids $appGatewaySubnetId -o tsv --query "id")

# set up bi-directional peering between aks and gateway vnet
az network vnet peering create -n gateway2aks \
    -g $appGatewayResourceGroup --vnet-name $appGatewayVnetName \
    --remote-vnet $aksVnetId \
    --allow-vnet-access
az network vnet peering create -n aks2gateway \
    -g $nodeResourceGroup --vnet-name $aksVnetName \
    --remote-vnet $appGatewayVnetId \
    --allow-vnet-access
```

If you are using `Azure CNI` for network plugin with AKS, then you are good to go.

If you are using `Kubenet` network plugin, then jump to [Kubenet](#with-kubenet) to setup the route table.

## Network Plugin used with AKS

### With Azure CNI
When using Azure CNI, Every pod is assigned a VNET route-able private IP from the subnet. So, Gateway should be able reach the pods directly.

### With Kubenet
When using Kubenet mode, Only nodes receive an IP address from subnet. Pod are assigned IP addresses from the `PodIPCidr` and a route table is created by AKS. This route table helps the packets destined for a POD IP reach the node which is hosting the pod.

When packets leave Application Gateway instances, Application Gateway's subnet need to aware of these routes setup by the AKS in the route table.

A simple way to achieve this is by associating the same route table created by AKS to the Application Gateway's subnet. When AGIC starts up, it checks the AKS node resource group for the existence of the route table. If it exists, AGIC will try to assign the route table to the Application Gateway's subnet, given it doesn't already have a route table. If AGIC doesn't have permissions to any of the above resources, the operation will fail and an error will be logged in the AGIC pod logs.

This association can also be performed manually:

```bash
aksClusterName="<aksClusterName>"
aksResourceGroup="<aksResourceGroup>"
appGatewayName="<appGatewayName>"
appGatewayResourceGroup="<appGatewayResourceGroup>"

# find route table used by aks cluster
nodeResourceGroup=$(az aks show -n $aksClusterName -g $aksResourceGroup -o tsv --query "nodeResourceGroup")
routeTableId=$(az network route-table list -g $nodeResourceGroup --query "[].id | [0]" -o tsv)

# get the application gateway's subnet
appGatewaySubnetId=$(az network application-gateway show -n $appGatewayName -g $appGatewayResourceGroup -o tsv --query "gatewayIpConfigurations[0].subnet.id")

# associate the route table to Application Gateway's subnet
az network vnet subnet update \
--ids $appGatewaySubnetId
--route-table $routeTableId
```

 ### Further Readings
  - [Peer the two virtual networks together](https://docs.microsoft.com/en-us/azure/application-gateway/tutorial-ingress-controller-add-on-existing#peer-the-two-virtual-networks-together)
  - [Virtual network peering](https://docs.microsoft.com/en-us/azure/virtual-network/virtual-network-peering-overview)
  - [How to peer your networks from different subscription](https://docs.microsoft.com/en-us/azure/virtual-network/create-peering-different-subscriptions)
  - [Use kubenet to configure networking](https://docs.microsoft.com/en-us/azure/aks/configure-kubenet)
  - [Use CNI to configure networking](https://docs.microsoft.com/en-us/azure/aks/configure-azure-cni)
  - [Network concept for AKS and Kubernetes](https://docs.microsoft.com/en-us/azure/aks/concepts-network)
  - [When to decide to use kubenet or CNI](https://docs.microsoft.com/en-us/azure/aks/configure-kubenet#choose-a-network-model-to-use)
