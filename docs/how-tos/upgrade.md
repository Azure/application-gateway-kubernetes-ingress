# Upgrading your helm installation

1) Update the helm repo and check the available chart versions.

    ```bash
    $> helm repo update && helm search -l application-gateway-kubernetes-ingress
    NAME                                                    CHART VERSION   APP VERSION     DESCRIPTION                                                 
    application-gateway-kubernetes-ingress/ingress-azure    0.7.0-rc1       0.7.0-rc1       Use Azure Application Gateway as the ingress for an Azure...
    application-gateway-kubernetes-ingress/ingress-azure    0.6.0           0.6.0           Use Azure Application Gateway as the ingress for an Azure...
    application-gateway-kubernetes-ingress/ingress-azure    0.6.0-rc1       0.6.0-rc1       Use Azure Application Gateway as the ingress for an Azure...
    ...
    ```

2) Look up your helm chart release name

    ```bash
    $> helm list
    NAME            REVISION        UPDATED                         STATUS  CHART                   APP VERSION     NAMESPACE
    odd-billygoat   22              Fri Jun 21 15:56:06 2019        FAILED  ingress-azure-0.7.0-rc1 0.7.0-rc1       default
    ```

2) Now to upgrade to a new version, use

```bash
helm upgrade application-gateway-kubernetes-ingress/ingress-azure --version <version>
```

## Rollback

If for some reason, the new deployment of the ingres controller goes to a bad state, rollback to the previous version using

```bash
helm rollback
```
