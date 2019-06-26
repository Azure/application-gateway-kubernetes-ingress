# Upgrading AGIC using Helm

The Azure Application Gateway Ingress Controller for Kubernetes (AGIC) can be upgraded
using [a Helm repository hosted on GitHub](https://azure.github.io/application-gateway-kubernetes-ingress/helm/).

Before we begin the upgrade procedure, ensure that you have added the required repository:

- View your currently added Helm repositories with:
    ```bash
    helm repo list
    ```

- Add the AGIC repo with:
    ```bash
    helm repo add \
        application-gateway-kubernetes-ingress \
        https://azure.github.io/application-gateway-kubernetes-ingress/helm/
    ```


### Upgrade
1) Refresh the AGIC Helm repository to get the latest release:

    ```bash
    helm repo update
    ```

2) View available versions of the `application-gateway-kubernetes-ingress` chart:

    ```bash
    helm search -l application-gateway-kubernetes-ingress
    ```

    Sample response:
    ```bash
    NAME                                                    CHART VERSION   APP VERSION     DESCRIPTION
    application-gateway-kubernetes-ingress/ingress-azure    0.7.0-rc1       0.7.0-rc1       Use Azure Application Gateway as the ingress for an Azure...
    application-gateway-kubernetes-ingress/ingress-azure    0.6.0           0.6.0           Use Azure Application Gateway as the ingress for an Azure...
    ```

    Latest available version from the list above is: `0.7.0-rc1`

2) View the Helm charts currently installed:

    ```bash
    helm list
    ```

    Sample response:
    ```
    NAME            REVISION        UPDATED                         STATUS  CHART                   APP VERSION     NAMESPACE
    odd-billygoat   22              Fri Jun 21 15:56:06 2019        FAILED  ingress-azure-0.7.0-rc1 0.7.0-rc1       default
    ```

    The Helm chart installation from the sample response above is named `odd-billygoat`. We will
    use this name for the rest of the commands. Your actual deployment name will most likely differ.

2) Upgrade the Helm deployment to a new version:

    ```bash
    helm upgrade \
        application-gateway-kubernetes-ingress/ingress-azure \
        -n odd-billygoat \
        --version 0.7.0-rc1
    ```

## Rollback

Should the Helm deployment fail, you can rollback to a previous release.
1) Get the last known healthy release number:

    ```bash
    helm history odd-billygoat
    ```

    Sample output:

    ```
    REVISION        UPDATED                         STATUS          CHART                   DESCRIPTION
    1               Mon Jun 17 13:49:42 2019        DEPLOYED        ingress-azure-0.6.0     Install complete
    2               Fri Jun 21 15:56:06 2019        FAILED          ingress-azure-xx        xxxx
    ```

    From the sample output of the `helm history` command it looks like the last successful deployment of our `odd-billygoat` was revision `1`

2) Rollback to the last successful revision:

    ```bash
    helm rollback odd-billygoat 1
    ```
