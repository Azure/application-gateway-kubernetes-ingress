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
  - get logs with `kubectl logs <pod-name> -n <namespace>`
