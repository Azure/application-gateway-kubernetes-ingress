## Troubleshooting: AGIC pod stuck in not ready state

.. note::
    [Application Gateway for Containers](https://aka.ms/agc) has been released, which introduces numerous performance, resilience, and feature changes. Please consider leveraging Application Gateway for Containers for your next deployment.

### Illustration

If AGIC pod is stuck in ready state, you must be seeing the following:

```
$ kubectl get pods

NAME                                   READY   STATUS    RESTARTS   AGE
<AGIC-POD-NAME>                        0/1     Running   0          19s
mic-774b9c5d7b-z4z8p                   1/1     Running   1          15m
mic-774b9c5d7b-zrdsm                   1/1     Running   1          15m
nmi-pv8ch       
```

### Common causes:
1. [Stuck at creating authorizer](#agic-is-stuck-at-creating-authorizer)
1. [Stuck getting Application Gateway](#agic-is-stuck-getting-application-gateway)

### AGIC is stuck at creating authorizer
When the AGIC pod starts, in one of the steps, AGIC tries to get an AAD (Azure Active Directory) token for the identity assigned to it. This token is then used to perform updates on the Application gateway.  
This identity can be of two types:
1. User Assigned Identity
1. Service Principal

When using User Assigned identity with AGIC, AGIC has a dependency on [`AAD Pod Identity`](https://github.com/Azure/aad-pod-identity).  
When you see your AGIC pod stuck at `Creating Authorizer` step, then the issue could be related to the setup of the user assigned identity and AAD Pod Identity.

```
$ kubectl logs <AGIC-POD-NAME>
ERROR: logging before flag.Parse: I0628 18:09:49.947221       1 utils.go:115] Using verbosity level 3 from environment variable APPGW_VERBOSITY_LEVEL
I0628 18:09:49.987776       1 environment.go:240] KUBERNETES_WATCHNAMESPACE is not set. Watching all available namespaces.
I0628 18:09:49.987861       1 main.go:128] Application Gateway Details: Subscription="xxxx" Resource Group="resgp" Name="gateway"
I0628 18:09:49.987873       1 auth.go:46] Creating authorizer from Azure Managed Service Identity
I0628 18:09:49.987945       1 httpserver.go:57] Starting API Server on :8123
```

`AAD Pod Identity` is responsible for assigning the user assigned identity provided by the user for AGIC as `AGIC's Identity` to the underlying AKS nodes and setup the IP table rules to allow AGIC to get an AAD token from the Instance Metadata service on the VM. When you install `AAD Pod Identity` on your AKS cluster, it will deploy two components:
1. Managed Identity Controller (MIC): It runs with multiple replicas and one Pod is **elected leader**. It is responsible to do the assignment of the identity to the AKS nodes.
1. Node Managed Identity (NMI): It runs as **daemon on every node**. It is responsible to enforce the IP table rules to allow AGIC to `GET` the access token.

For further reading on how these components work, you can go through [this readme](https://github.com/Azure/aad-pod-identity#components). Here is a [concept diagram](https://github.com/Azure/aad-pod-identity/blob/master/docs/design/concept.png) on the project page.

Now, In order to debug the authorizer issue further, we need to get the logs for `mic` and `nmi` pods. These pods usually start with mic and nmi as the prefix. We should first investigate the logs of `mic` and then `nmi`.

```
$ kubectl get pods

NAME                                   READY   STATUS    RESTARTS   AGE
mic-774b9c5d7b-z4z8p                   1/1     Running   1          15m
mic-774b9c5d7b-zrdsm                   1/1     Running   1          15m
nmi-pv8ch                              1/1     Running   1          15m
```

###### Issue in MIC Pod
For `mic` pod, we will need to find the leader. An easy way to find the leader is by looking at the log size. Leader pod is the one that is actively working.
1. MIC pod communicates with Azure Resource Manager(ARM) to assign the identity to the AKS nodes. If there are any issues in outbound connectivity, MIC can report TCP timeouts. Check your NSGs, UDRs and Firewall to make sure that you allow outbound traffic to Azure.
    ```
    Updating msis on node aks-agentpool-41724381-vmss, add [1], del [1], update[0] failed with error azure.BearerAuthorizer#WithAuthorization: Failed to refresh the Token for request to https://management.azure.com/subscriptions/xxxx/resourceGroups/resgp/providers/Microsoft.Compute/virtualMachineScaleSets/aks-agentpool-41724381-vmss?api-version=2019-07-01: StatusCode=0 -- Original Error: adal: Failed to execute the refresh request. Error = 'Post "https://login.microsoftonline.com/<tenantId>/oauth2/token?api-version=1.0": dial tcp: i/o timeout'
    ```
1. You will see the following error if AKS cluster's `Service Principal` missing `Managed Identity Operator` access over User Assigned identity. You can follow the role assignment related step in the [brownfield document](https://github.com/Azure/application-gateway-kubernetes-ingress/blob/master/docs/setup/install-existing.md#set-up-aad-pod-identity).
    ```
    Updating msis on node aks-agentpool-32587779-vmss, add [1], del [0] failed with error compute.VirtualMachineScaleSetsClient#CreateOrUpdate: Failure sending request: StatusCode=403 -- Original Error: Code="LinkedAuthorizationFailed" Message="The client '<objectID>' with object id '<objectID>' has permission to perform action 'Microsoft.Compute/virtualMachineScaleSets/write' on scope '/subscriptions/xxxx/resourceGroups/<nodeResourceGroup>/providers/Microsoft.Compute/virtualMachineScaleSets/aks-agentpool-32587779-vmss'; however, it does not have permission to perform action 'Microsoft.ManagedIdentity/userAssignedIdentities/assign/action' on the linked scope(s) '/subscriptions/xxxx/resourcegroups/resgp/providers/Microsoft.ManagedIdentity/userAssignedIdentities/<identityName>' or the linked scope(s) are invalid."
    ```

###### Issue in NMI Pod
For `nmi` pod, we will need to find the pod running on the same node as AGIC pod.
1. If you see `403` response for a token request, then make sure you have correctly assigned the needed permission to `AGIC's identity`.
    1. `Reader` access to Application Gateway's resource group. This is needed to list the resources in the this resource group.
    1. `Contributor` access to Application Gateway. This is needed to perform updates on the Application Gateway.

### AGIC is stuck getting Application Gateway
AGIC can be stuck in getting the gateway due to:
1. AGIC gets `NotFound` when getting Application Gateway  
When you see this error,
    1. Verify that the gateway actually exists in the subscription and resource group printed in the AGIC logs.
    1. If you are deploying in National Cloud or US Gov Cloud, then this issue could be related to incorrect environment endpoint setting. To correctly configure, set the [`appgw.environment`](../helm-values-documenation.md) property in the helm.
1. AGIC gets `Unauthorized` when getting Application Gateway  
Verify that you have given needed permissions to AGIC's identity:
    1. `Reader` access to Application Gateway's resource group. This is needed to list the resources in the this resource group.
    1. `Contributor` access to Application Gateway. This is needed to perform updates on the Application Gateway.
