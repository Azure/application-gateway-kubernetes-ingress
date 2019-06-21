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
    
    Latest Version (current): `0.7.0-rc1`

2) Look up your helm chart release name

    ```bash
    $> helm list
    NAME            REVISION        UPDATED                         STATUS  CHART                   APP VERSION     NAMESPACE
    odd-billygoat   22              Fri Jun 21 15:56:06 2019        FAILED  ingress-azure-0.7.0-rc1 0.7.0-rc1       default
    ```

    Release name: `odd-billygoat`

2) Now to upgrade to a new version, use

    ```bash
    $> helm upgrade application-gateway-kubernetes-ingress/ingress-azure -n odd-billygoat --version 0.7.0-rc1
    ```

## Rollback

If for some reason, the new deployment of the ingres controller goes to a bad state.
Check your previous revision number by,

```bash
$> helm history odd-billygoat
REVISION        UPDATED                         STATUS          CHART                   DESCRIPTION                                                 
1               Mon Jun 17 13:49:42 2019        DEPLOYED      ingress-azure-0.6.0     Install complete                                            
2              Fri Jun 21 15:56:06 2019        FAILED          ingress-azure-xx    xxxx
```

Rollback using:

```bash
$> helm rollback odd-billygoat 1
Rollback was a success! Happy Helming!
```
