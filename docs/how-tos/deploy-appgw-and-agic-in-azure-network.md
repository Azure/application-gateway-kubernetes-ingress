## Overview

AGIC(Application Gateway Ingress Controllor) is a pod running in AKS/Kubernetes cluster, to make AGIC work properly with AppGw(Azure Application Gateway), there are multiple ways to deploy them in Azure network:
  - AGIC and AppGw are deployed in the same network
  - AGIC and AppGw are deployed in different networks

## Deploy in the same vnet
AGIC as a pod running in AKS can be deployed in the same network as AppGw, there are two different network connectivities,  kubenet (basic networking) and Azure CNI (advanced networking).
### With kubenet
only the nodes receive an IP address in the virtual network subnet. route table is needed for connectivity between AGIC and AppGw nodes if it doesn't exist. 
Note that If your custom subnet doesn’t contain a route table, AKS creates one for you and adds rules to it. If your custom subnet contains a route table when you create your cluster, AKS acknowledges the existing route table during cluster operations and updates rules accordingly for cloud provider operations. more reading can be found [Bring your own subnet and route table with kubenet](https://docs.microsoft.com/en-us/azure/aks/configure-kubenet#bring-your-own-subnet-and-route-table-with-kubenet)

### With CNI
every pod gets an IP address from the node subnet and can be accessed directly with the IP.

Further readings:
  - [Use kubenet to configure networking](https://docs.microsoft.com/en-us/azure/aks/configure-kubenet)
  - [Use CNI to configure networking](https://docs.microsoft.com/en-us/azure/aks/configure-azure-cni)
  - [Network concept for AKS and Kubernetes](https://docs.microsoft.com/en-us/azure/aks/concepts-network)
  - [When to decide to use kubenet or CNI](https://docs.microsoft.com/en-us/azure/aks/configure-kubenet#choose-a-network-model-to-use)


## Deploy in the different vnets
AGIC pod(or AKS) can be deployed to different vNet from AppGw's vNet, however, the two vNets must be [peered](https://docs.microsoft.com/en-us/azure/virtual-network/virtual-network-peering-overview) together. i.e. AKS cluster can be configured on a virtual network peered with an AppGw.
In both CNI and Kubenet, UDR(User-Defined Route) is needed to configure the next hop of AKS egress to virtual appliance such as Application Gateway.

 Further readings:
  - [Customize AKS cluster egress with a User-Defined Route](https://docs.microsoft.com/en-us/azure/aks/egress-outboundtype)