# Troubleshooting

Application Gateway Ingress Controller (AGIC) continuously monitors the folowing Kubernetes resources:
  - [Deployment](https://kubernetes.io/docs/concepts/workloads/controllers/deployment/#creating-a-deployment) or [Pod](https://kubernetes.io/docs/concepts/workloads/pods/pod/#what-is-a-pod)
  - [Service](https://kubernetes.io/docs/concepts/services-networking/service/)
  - [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/)

The following must be in place for AGIC to configure App Gateway with IPs of Kubernetes pods:
  1. One or more healthy pods
  2. One or more services, referencing the pods above via matching `selector` labels
  3. Ingress, annotated with `kubernetes.io/ingress.class: azure/application-gateway`, referencing the service above

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


* The AGIC pod should be in the `default` namespace (see column `NAMESPACE`). A healthy pod would have `Running` in the `STATUS` column. There should be at least one AGIC pod.
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


* Is your [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) annotated with: `kubernetes.io/ingress.class: azure/application-gateway`? AGIC will only watch for Kubernetes Ingress resources that have this annotation.
```bash
# Get the YAML definition of a particular ingress resource
kubectl get ingress --namespace  <which-namespace?>  <which-ingress?>  -o yaml
```


* AGIC emits Kubernetes events for certain critical errors. You can view these:
  - in your terminal via `kubectl get events --sort-by=.metadata.creationTimestamp`
  - in your browser using the [Kubernetes Web UI (Dashboard)](https://kubernetes.io/docs/tasks/access-application-cluster/web-ui-dashboard/)


# Logging Levels

AGIC has 3 logging levels. Level 1 is the default one and it shows minimal number of log lines.
Level 5, on the other hand, would display all logs, including sanitized contents of config applied
to ARM.

The Kubernetes community has established 9 levels of logging for
the [kubectl](https://kubernetes.io/docs/reference/kubectl/cheatsheet/#kubectl-output-verbosity-and-debugging) tool. In this
repository we are utilizing 3 of these, with similar semantics:


| Verbosity | Description |
|-----------|-------------|
|  1        | Default log level; shows startup details, warnings and errors |
|  3        | Extended information about events and changes; lists of created objects |
|  5        | Logs marshaled objects; shows sanitized JSON config applied to ARM |


The verbosity levels are adjustable via the `verbosityLevel` variable in the
[helm-config.yaml](examples/sample-helm-config.yaml) file. Increase verbosity level to `5` to get
the JSON config dispatched to
[ARM](https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-overview):
  - add `verbosityLevel: 5` on a line by itself in [helm-config.yaml](examples/sample-helm-config.yaml) and re-install
  - get logs with `kubectl logs <pod-name>`
