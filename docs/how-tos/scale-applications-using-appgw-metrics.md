# Scale your Applications using Application Gateway Metrics (Beta)

As incoming traffic increases, it becomes crucial to scale up your applications based on the demand.

In the following tutorial, we explain how you can use Application Gateway's `AvgRequestCountPerHealthyHost` metric to scale up your application. `AvgRequestCountPerHealthyHost` is measure of average request that are sent to a specific backend pool and backend http setting combination.

We are going to use following two components:

1. [`Azure K8S Metric Adapter`](https://github.com/Azure/azure-k8s-metrics-adapter) - We will using the metric adapter to expose Application Gateway metrics through the metric server.
1. [`Horizontal Pod Autoscaler`](https://docs.microsoft.com/en-us/azure/aks/concepts-scale#horizontal-pod-autoscaler) - We will use HPA to use Application Gateway metrics and target a deployment for scaling.

## Setting up Azure K8S Metric Adapter

1. We will first create an Azure AAD service principal and assign it `Monitoring Reader` access over Application Gateway's resource group. Paste the following lines in your [Azure Cloud Shell](https://shell.azure.com/):
    ```bash
    applicationGatewayGroupName="<application-gateway-group-id>"
    applicationGatewayGroupId=$(az group show -g $applicationGatewayGroupName -o tsv --query "id")
    az ad sp create-for-rbac -n "azure-k8s-metric-adapter-sp" --role "Monitoring Reader" --scopes applicationGatewayGroupId
    ```

1. Now, We will deploy the [`Azure K8S Metric Adapter`](https://github.com/Azure/azure-k8s-metrics-adapter) using the AAD service principal created above.

    ```bash
    kubectl create namespace custom-metrics

    # use values from service principle created above to create secret
    kubectl create secret generic azure-k8s-metrics-adapter -n custom-metrics \
        --from-literal=azure-tenant-id=<tenantid> \
        --from-literal=azure-client-id=<clientid> \
        --from-literal=azure-client-secret=<secret>

    kubectl apply -f kubectl apply -f https://raw.githubusercontent.com/Azure/azure-k8s-metrics-adapter/master/deploy/adapter.yaml -n custom-metrics
    ```

1. We will create an `ExternalMetric` resource with name `appgw-request-count-metric`. This will instruct the metric adapter to expose `AvgRequestCountPerHealthyHost` metric for `myApplicationGateway` resource in `myResourceGroup` resource group. You can use the `filter` field to target a specific backend pool and backend http setting in the Application Gateway. Copy paste this YAML content in `external-metric.yaml` and apply with `kubectl apply -f external-metric.yaml`.

    ```yaml
    apiVersion: azure.com/v1alpha2
    kind: ExternalMetric
    metadata:
    name: appgw-request-count-metric
    spec:
      type: azuremonitor
      azure:
        resourceGroup: myResourceGroup # replace with your application gateway's resource group name
        resourceName: myApplicationGateway # replace with your application gateway's name
        resourceProviderNamespace: Microsoft.Network
        resourceType: applicationGateways
      metric:
        metricName: AvgRequestCountPerHealthyHost
        aggregation: Average
        filter: BackendSettingsPool eq '<backend-pool-name>~<backend-http-setting-name>' # optional
    ```

You can now make a request to the metric server to see if our new metric is getting exposed:
```bash
kubectl get --raw "/apis/external.metrics.k8s.io/v1beta1/namespaces/default/appgw-request-count-metric"
# Sample Output
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

## Using the new metric to scale up our deployment

Once we are able to expose `appgw-request-count-metric` through the metric server, We are ready to use [`Horizontal Pod Autoscaler`](https://docs.microsoft.com/en-us/azure/aks/concepts-scale#horizontal-pod-autoscaler) to scale up our target deployment.

In following example, we will target a sample deployment `aspnet`. We will scale up Pods when `appgw-request-count-metric` > 200 per Pod upto a max of `10` Pods.

Replace your target deployment name and apply the following auto scale configuration. Copy paste this YAML content in `autoscale-config.yaml` and apply with `kubectl apply -f autoscale-config.yaml`.
```yaml
apiVersion: autoscaling/v2beta1
kind: HorizontalPodAutoscaler
metadata:
  name: deployment-scaler
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: aspnet # replace with your deployment's name
  minReplicas: 1
  maxReplicas: 10
  metrics:
  - type: External
    external:
      metricName: appgw-request-count-metric
      targetAverageValue: 200
```

Test your configuration by using a load test tools like apache bench:
```bash
ab -n10000 http://<application-gateway-ip-address>/
```