# Instructions: Installing a Sample HTTPS App

## Prerequisites

1. AKS cluster with networking plugin `azure` and AGIC installed
2. [Application Gateway](https://azure.microsoft.com/en-us/services/application-gateway/) in the same [VNet](https://docs.microsoft.com/en-us/azure/virtual-network/virtual-networks-overview)

## Steps

1) Get cluster credentials

    ```bash
    az aks get-credentials \
        --resource-group <rg> \
        --name <aks>
    ```

    Where:
      - `<rg>` is the Resource Group within which the AKS cluster is installed
      - `<aks>` is the name of the AKS cluster.

2) Deploy the application on k8s cluster and expose as a service internally.

    ```bash
    kubectl apply -f deployment.yaml
    kubectl apply -f service.yaml
    ```

    Where [deployment.yaml](deployment.yaml) and [service.yaml](service.yaml) are files in the current directory of this repository.

3) Add ingress rule

    ```bash
    kubectl apply -f sample-app-ingress-http.yaml
    ```

    Navigate to https://portal.azure.com and view the endpoints of your Application Gateway to verify that the Service is correctly exposed.

4) Create a new self-signed certificate:

    ```bash
    openssl req -x509 -nodes \
        -days 365 \
        -newkey rsa:2048 \
        -out sample-app-tls.crt \
        -keyout sample-app-tls.key \
        -subj "/CN=sample-app"
    ```

5) Create a Kubernetes secret with the certificate generated above:

    ```bash
    kubectl create secret tls sample-app-tls \
        --key sample-app-tls.key \
        --cert sample-app-tls.crt
    ```

6) Update ingress rule to use TLS

    ```bash
    kubectl apply -f sample-app-ingress-https.yaml
    ```

    Where [sample-app-ingress-https.yaml](sample-app-ingress-https.yaml) is a file in the current directory of this repository.

    Verify that the endpoint is now HTTPS enabled.
