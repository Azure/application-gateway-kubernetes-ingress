# Troubleshooting


The Application Gateway Ingress Controller relies primarily on the Kubernetes
[Service](https://kubernetes.io/docs/concepts/services-networking/service/) and
[Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) resources to construct
configuration for App Gateway. Surprising AGIC behavior (or none at all) could be as a result of
missing or incorrect configuration.


* Get the existing namespaces in Kubernetes cluster. What namespace is your app
running in? Is AGIC watching that namespace? Refer to the
[Multiple Namespace Support](features/multiple-namespaces.md#enable-multiple-namespace-support)
documentation on how to properly configure observed namespaces.
```bash
# What namespaces exist on your cluster
kubectl get namespaces

# What pods are currently running
kubectl get pods --all-namespaces -o wide
```

* The AGIC pod should be in the `default` namespace (see column `NAMESPACE`). A healthy pod would have `Running` in the `STATUS` column. There should be exactly one AGIC pod.
```bash
# Get a list of the Application Gateway Ingress Controller pods
kubectl get pods --all-namespaces --selector app=ingress-azure
```


* If the AGIC pod is not healthy (`STATUS` column from the command above is not `Running`):
  - get logs to understand why: `kubectl logs <pod-name>`
  - for the previous instance of the pod: `kubectl logs <pod-name> --previous`
  - describe the pod to get more context: `kubectl describe pod <pod-name>`


* Do you have a Kubernetes
[Service](https://kubernetes.io/docs/concepts/services-networking/service/) and
[Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) resources?
```bash
# Get all services across all namespaces
kubectl get service --all-namespaces -o wide

# Get all ingress resources across all namespaces
kubectl get ingress --all-namespaces -o wide
```


* Is your [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) annotated with: `kubernetes.io/ingress.class: azure/application-gateway`
```bash
# Get the YAML definition of a particular ingress resource
kubectl get ingress --namespace  <which-namespace?>  <which-ingress?>  -o yaml
```

* AGIC has verbose logging capability. It is not enabled by default. This is controlled via an environment variable. Increase the **verbosity level** of AGIC to `5` to get the JSON config
dispatched to [ARM](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-overview):
  - add `verbosityLevel: 5` on a line by itself in [helm-config.yaml](examples/sample-helm-config.yaml) and re-install
  - get logs with `kubectl logs <pod-name>`


* AGIC emits Kubernetes events for certain critical errors. You can view these:
  - in your terminal via `kubectl get events --sort-by=.metadata.creationTimestamp`
  - in your browser using the [Kubernetes Web UI (Dashboard)](https://kubernetes.io/docs/tasks/access-application-cluster/web-ui-dashboard/)
