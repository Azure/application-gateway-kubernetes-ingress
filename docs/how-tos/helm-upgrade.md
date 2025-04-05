# Upgrading AGIC using Helm

> **_NOTE:_** [Application Gateway for Containers](https://aka.ms/agc) has been released, which introduces numerous performance, resilience, and feature changes. Please consider leveraging Application Gateway for Containers for your next deployment.

The Azure Application Gateway Ingress Controller for Kubernetes (AGIC) can be upgraded
using a Helm repository hosted on MCR.

## Upgrade

1. View the Helm charts currently installed:

    ```bash
    helm list
    ```

    Sample response:

    ```bash
    NAME            REVISION        UPDATED                         STATUS  CHART                   APP VERSION     NAMESPACE
    odd-billygoat   22              Fri Nov 08 15:56:06 2019        FAILED  ingress-azure-1.0.0     1.0.0           default
    ```

    The Helm chart installation from the sample response above is named `odd-billygoat`. We will
    use this name for the rest of the commands. Your actual deployment name will most likely differ.

1. Upgrade the Helm deployment to a new version:

    ```bash
    helm upgrade \
        odd-billygoat \
        oci://mcr.microsoft.com/azure-application-gateway/charts/ingress-azure \
        --version 1.8.0
    ```

## Rollback

Should the Helm deployment fail, you can rollback to a previous release.

1. Get the last known healthy release number:

    ```bash
    helm history odd-billygoat
    ```

    Sample output:

    ```bash
    REVISION        UPDATED                         STATUS          CHART                   DESCRIPTION
    1               Mon Jun 17 13:49:42 2019        DEPLOYED        ingress-azure-0.6.0     Install complete
    2               Fri Jun 21 15:56:06 2019        FAILED          ingress-azure-xx        xxxx
    ```

    From the sample output of the `helm history` command it looks like the last successful deployment of our `odd-billygoat` was revision `1`

1. Rollback to the last successful revision:

    ```bash
    helm rollback odd-billygoat 1
    ```
