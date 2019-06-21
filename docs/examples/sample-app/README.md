# Instructions

## Pre-requisite:

Application Gateway and AKS cluster.

## Steps

1) Get cluster credentials

    ```bash
    az aks get-credentials --resource-group <rg> --name <aks>
    ```

2) Deploy the application on k8s cluster and expose as a service internally.

    ```bash
    kubectl apply -f deployment.yaml
    kubectl apply -f service.yaml
    ```

3) Add ingress rule

    ```bash
    kubectl apply -f sample-app-ingress-http.yaml
    ```

    Now browse the Application Gateway endpoint to check that the Service is exposed.

4) Create a self-signed certificate

    ```bash
    openssl req -x509 -nodes -days 365 -newkey rsa:2048 -out sample-app-tls.crt -keyout sample-app-tls.key -subj "/CN=sample-app"
    ```

5) Create a k8s secret with the certificate

    ```bash
    kubectl create secret tls sample-app-tls --key sample-app-tls.key --cert sample-app-tls.crt
    ```

6) Update ingress rule to use TLS

    ```bash
    kubectl apply -f sample-app-ingress-https.yaml
    ```

    Visit the same endpoint with https.
