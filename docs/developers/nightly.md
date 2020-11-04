# Install the latest nightly build

To install the latest nightly release,

1. Add the nightly helm repository
    ```
    helm repo add agic-nightly https://appgwingress.blob.core.windows.net/ingress-azure-helm-package-staging/
    helm repo update
    ```

2. Check the available version

    **Latest version**: ![nightly release (latest by date)](https://img.shields.io/badge/dynamic/yaml?url=https://appgwingress.blob.core.windows.net/ingress-azure-helm-package-staging/index.yaml&label=nightly&query=entries[%22ingress-azure%22][0].appVersion&color=green)  
    
    or  
    
    You can look up the version in the repo using helm.
    ```
    helm search repo agic-nightly
    ```

2.  Install using the same helm command by using the staging repository.
    ```
    helm install ingress-azure \
      -f helm-config.yaml \
      agic-nightly/ingress-azure \
      --version <version>
    ```