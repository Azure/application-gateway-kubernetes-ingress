## Overview

AGIC (Application Gateway Ingress Controllor) is a pod running in AKS/Kubernetes cluster, to make AGIC work properly with AppGw (Azure Application Gateway), there are multiple ways to deploy them in Azure network:
  - AGIC and AppGw are deployed in the same network
  - AGIC and AppGw are deployed in different networks

## Deploy in the same network
AGIC as a pod running in AKS can be deployed in the same network as AppGw, there are two different network connectivities,  kubenet (basic networking) and Azure CNI (advanced networking).
With kubenet, only the nodes receive an IP address in the virtual network subnet. Pods can't communicate directly with node outside of kubernetes cluster. With CNI, every pod gets an IP address from the node subnet and can be accessed directly with the IP. In both cases, User Defined Routing (UDR) and IP forwarding is needed for connectivity between AGIC pod and AppGw instances.

Further readings:
  - [Use kubenet to configure networking](https://docs.microsoft.com/en-us/azure/aks/configure-kubenet)
  - [Use CNI to configure networking](https://docs.microsoft.com/en-us/azure/aks/configure-azure-cni)
  - [Network concept for AKS and Kubernetes](https://docs.microsoft.com/en-us/azure/aks/concepts-network)
  - [Customize cluster egress with a User-Defined Route](https://docs.microsoft.com/en-us/azure/aks/egress-outboundtype)