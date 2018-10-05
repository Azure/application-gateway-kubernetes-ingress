# Content
- [Creating Application Gateway and Azure Kubernetes Cluster using Azure Resource Manager (ARM) template](#deploying-the-infrastructure)
- [Setting up the Azure Kubernetes Cluster](#setting-up-the-azure-kubernetes-cluster)

## Deploying the infrastructure

To create the pre-requisite Azure resources, You can use the following template. It creates:
1) Azure Virtual Network with 2 subnets.
2) Azure Application Gateway v2
3) Azure Kubernetes Service cluster with required permission to deploy nodes in the vnet.
4) User Assigned Identity to initialize the aad-pod-identity service and ingress controller.
5) Set required RBACs.


Steps:

1) Create a service principal that will be assigned to aks cluster in the template.
    ```
    az ad sp create-for-rbac --skip-assignment
    Note: appId and password.
    az ad sp show --id <appId>
    Note: objectId.
    ```

2) After the above, click to create a custom template deployment. Provide the appId for servicePrincipalClientId, password, objectId in the parameters.

<a href="https://portal.azure.com/#create/Microsoft.Template/uri/https%3A%2F%2Fraw.githubusercontent.com%2Fakshaysngupta%2Fapplication-gateway-kubernetes-ingress%2Fmaster%2Fdeploy%2Fazuredeploy.json" target="_blank">
    <img src="http://azuredeploy.net/deploybutton.png"/>
</a>
<a href="http://armviz.io/#/?load=https%3A%2F%2Fraw.githubusercontent.com%2Fakshaysngupta%2Fapplication-gateway-kubernetes-ingress%2Fmaster%2Fdeploy%2Fazuredeploy.json" target="_blank">
    <img src="http://armviz.io/visualizebutton.png"/>
</a>

After the deployment completes, you can look at the Output window for parameters needed in the following steps.

## Setting up the Azure Kubernetes Cluster

Once, we have the azure resources created, we need to deploy following pods on the cluster:
1) `aad pod identity` - This controller uses user assigned identity and provide ARM access token to the controller.
2) `application gateway ingress controller` - This controller communicates ingress realted events to the Application Gateway resource.

Steps:

1) Get credentials for the newly created Azure Kubernetes Cluster. This will cache the credentials n kubeconfig and set the context.  
    `az aks get-credentials --resource-group <rg> --name <aksClusterName>`

2) Add aad pod idenitty service to the cluster using the following command. This service will be used  by controller . You can refer [aad-pod-identity](https://github.com/Azure/aad-pod-identity).  
    `kubectl create -f https://raw.githubusercontent.com/Azure/aad-pod-identity/master/deploy/infra/deployment.yaml`

3) Create the ingress controller in the cluster. This ingress controller will communicate updates the applcation gateway.
    ```
    helm init
    helm repo add application-gateway-kubernetes-ingress https://azure.github.io/application-gateway-kubernetes-ingress/helm/
    helm repo update
    ```

4) Edit [helm-config.yaml](example/helm-config.yaml) and fill in the values for `appgw` and `armAuth`
    ```yaml
    # This file contains the essential configs for the ingress controller helm chart

    ################################################################################
    # Specify which application gateway the ingress controller will manage
    #
    appgw:
        subscriptionId: <subscription-id>
        resourceGroup: <resourcegroup-name>
        name: <applicationgateway-name>

    ################################################################################
    # Specify which kubernetes namespace the ingress controller will watch
    # Default value is "default"
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
        identityResourceID: <identity-resource-id. You can refer the template deployment output.>
        identityClientID:  <identity-client-id. You can refer the template deployment output.>
    ```

    Then execute the following to the install the Applicaiton Gateway ingress controller package.  
    `helm install -f helm-config.yaml application-gateway-kubernetes-ingress/ingress-azure`