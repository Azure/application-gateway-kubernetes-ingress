## Overview

AGIC(Application Gateway Ingress Controllor) is a pod running in AKS/Kubernetes cluster, to make AGIC work properly with AppGw(Azure Application Gateway), there are multiple ways to deploy them in Azure network:
  - AGIC and AppGw are deployed in the same virtual network
  - AGIC and AppGw are deployed in different virtual networks

## Deploy in same vnet
AGIC as a pod running in AKS can be deployed in the same network as AppGw, there are two different network connectivities,  kubenet (basic networking) and Azure CNI (advanced networking).
### With kubenet
only the nodes receive an IP address in the virtual network subnet. A route table will be created automatically for the connectivity between AGIC and the subnet where AppGw nodes are in.
Note that if your custom subnet contains a route table when you create your cluster, the custom route table must be associated to the subnet before you create the AKS cluster. more reading can be found [Bring your own subnet and route table with kubenet](https://docs.microsoft.com/en-us/azure/aks/configure-kubenet#bring-your-own-subnet-and-route-table-with-kubenet)

### With CNI
every pod gets an IP address from the node subnet and can be accessed directly with the IP, the pod can talk to AppGw nodes directly.

### Further Readings
  - [Use kubenet to configure networking](https://docs.microsoft.com/en-us/azure/aks/configure-kubenet)
  - [Use CNI to configure networking](https://docs.microsoft.com/en-us/azure/aks/configure-azure-cni)
  - [Network concept for AKS and Kubernetes](https://docs.microsoft.com/en-us/azure/aks/concepts-network)
  - [When to decide to use kubenet or CNI](https://docs.microsoft.com/en-us/azure/aks/configure-kubenet#choose-a-network-model-to-use)


## Deploy in different vnets
AGIC pod(or AKS) can be deployed in different vNet from AppGw's vNet, however, the two vNets must be [peered](https://docs.microsoft.com/en-us/azure/virtual-network/virtual-network-peering-overview) together. When you create a virtual network peering between two virtual networks, a route is added by Azure for each address range within the address space of each virtual network a peering is created for.

```bash
# example to peer appgw vnet and aks vnet 
nodeResourceGroup=$(az aks show -n myCluster -g myResourceGroup -o tsv --query "nodeResourceGroup")
aksVnetName=$(az network vnet list -g $nodeResourceGroup -o tsv --query "[0].name")

aksVnetId=$(az network vnet show -n $aksVnetName -g MC_$nodeResourceGroup -o tsv --query "id")
az network vnet peering create -n AppGWtoAKSVnetPeering -g myResourceGroup --vnet-name myVnet --remote-vnet $aksVnetId --allow-vnet-access

appGWVnetId=$(az network vnet show -n myVnet -g myResourceGroup -o tsv --query "id")
az network vnet peering create -n AKStoAppGWVnetPeering -g $nodeResourceGroup --vnet-name $aksVnetName --remote-vnet $appGWVnetId --allow-vnet-access
```

Note that when AKS created with `--network-plugin kubenet`, make sure the route table and NSG created in the MC_* resource group are associated to the Application Gateway subnet.

```bash
# find Route table and NSG     
AKS_MC_RG=$(az group list --query "[?starts_with(name, 'MC_${AKS_RG}')].name | [0]" --output tsv)
ROUTE_TABLE=$(az network route-table list -g ${AKS_MC_RG} --query "[].id | [0]" -o tsv)
AKS_NODE_NSG=$(az network nsg list -g ${AKS_MC_RG} --query "[].id | [0]" -o tsv)
APPGW_SUBNET_ID=$(az network vnet subnet show -g ${APPGW_VNET_RG} --name ${APPGW_SUBNET_NAME} --vnet-name ${APPGW_VNET_NAME} --query id -o tsv)

# Update the Appgw subnet
az network vnet subnet update \
-g $APPGW_VNET_RG \
--route-table $ROUTE_TABLE \
--network-security-group $AKS_NODE_NSG \
--ids $APPGW_SUBNET_ID
```

 ### Further Readings
  - [Peer the two virtual networks together](https://docs.microsoft.com/en-us/azure/application-gateway/tutorial-ingress-controller-add-on-existing#peer-the-two-virtual-networks-together)
  - [Virtual network peering](https://docs.microsoft.com/en-us/azure/virtual-network/virtual-network-peering-overview)
  - [How to peer your networks from different subscription](https://docs.microsoft.com/en-us/azure/virtual-network/create-peering-different-subscriptions)
