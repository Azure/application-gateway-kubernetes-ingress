# Scale your Applications using Application Gateway Metrics (Beta)

As incoming traffic increases, it becomes crucial to scale up your applications based on the demand.

In the following tutorial, we explain how you can use Application Gateway's `AvgRequestCountPerHealthyHost` metric to scale up your application. `AvgRequestCountPerHealthyHost` is measure of average reqeust that are sent to backend pool

We are going to use following two components:

1. [Azure K8S Metric Adapter](https://github.com/Azure/azure-k8s-metrics-adapter) - We will using the metric adapter to expose Applicaiton Gateway metrics through the metric server.
1. [Horizontal Pod Autoscaler](https://docs.microsoft.com/en-us/azure/aks/concepts-scale#horizontal-pod-autoscaler) - We will use HPA to use Applicaiton Gateway metrics and target a deployment.

## Setting up Azure K8S Metric Adapter

1. First, let's create an Azure AAD service principal which has access to the metrics for Application Gateway

    ```bash
    applicationGatewayGroupName="<application-gateway-group-id>"
    applicationGatewayGroupId=$(az group show -g $applicationGatewayGroupName -o tsv --query "id")
    az ad sp create-for-rbac -n "azure-k8s-metric-adapter-sp" --role "Monitoring Reader" --scopes applicationGatewayGroupId

    # use values from service principle created above to create secret
    kubectl create secret generic azure-k8s-metrics-adapter -n custom-metrics \
        --from-literal=azure-tenant-id=<tenantid> \
        --from-literal=azure-client-id=<clientid> \
        --from-literal=azure-client-secret=<secret>
    ```

1. Now create the following k8s resource to tell metric adapter to get metric for application gateway

    ```yaml
    apiVersion: azure.com/v1alpha2
    kind: ExternalMetric
    metadata:
    name: appgw-request-count-metric
    spec:
      type: azuremonitor
      azure:
        resourceGroup: <resource-group-name>
        resourceName: <application-gateway-name>
        resourceProviderNamespace: Microsoft.Network
        resourceType: applicationGateways
      metric:
        metricName: AvgRequestCountPerHealthyHost
        aggregation: Average
        filter: BackendSettingsPool eq '<backend-pool-name>~<backend-http-setting-name>' # optional
    ```

You can now test the metric by using:
```bash
kubectl get --raw "/apis/external.metrics.k8s.io/v1beta1/namespaces/default/appgw-request-count-metric"
# Output
# {
#   "kind": "ExternalMetricValueList",
#   "apiVersion": "external.metrics.k8s.io/v1beta1",
#   "metadata":
#     {
#       "selfLink": "/apis/external.metrics.k8s.io/v1beta1/namespaces/default/appgw-request-count-metric",
#     },
#   "items":
#     [
#       {
#         "metricName": "appgw-request-count-metric",
#         "metricLabels": null,
#         "timestamp": "2019-11-05T00:18:51Z",
#         "value": "30",
#       },
#     ],
# }
```

## Using the metric to scale up deployment

Now we will use the `appgw-request-count-metric` to scale up our deployment.

Fill in the `<deployment-name>` and create the following autoscale configuration.

```yaml
apiVersion: autoscaling/v2beta1
kind: HorizontalPodAutoscaler
metadata:
  name: deployment-scaler
spec:
  scaleTargetRef:
    apiVersion: extensions/v1beta1
    kind: Deployment
    name: <deployment-name>
  minReplicas: 1
  maxReplicas: 10
  metrics:
  - type: External
    external:
      metricName: appgw-request-count-metric
      targetAverageValue: 200
```

We are targetting `aspnet` deployment and we want to scale it when `targetAverageValue > 200` with an upper ceiling of max replicas being `10`. That is, when the numbers of requests is higher than 200 per Pod, we want to add 1 more pod until we reach 10 pods.