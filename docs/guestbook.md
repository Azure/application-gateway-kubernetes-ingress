# Example: Guestbook

This example will show you how to deploy a multi-tier web application and use a simple ingress to configure the application gateway. 
This example will also demonstrate how to set up TLS on the same service.

## Prerequisite

- Installed `ingress-azure` helm chart (see [here](install))
- If you want to use HTTPS on this application, you will need a x509 certificate and its private key.

## Deploy `guestbook` application

1. Download `guestbook-all-in-one.yaml` from [here](https://raw.githubusercontent.com/kubernetes/examples/master/guestbook/all-in-one/guestbook-all-in-one.yaml)
2. Deploy `guestbook-all-in-one.yaml` into your AKS cluster by running

    ```bash
    kubectl apply -f guestbook-all-in-one.yaml
    ```

Now, the `guestbook` application has been deployed.

By default, `guestbook` exposes its application through a service with name `frontend` on port `80`.

## Deploy Ingress without HTTPS

We will be using [ing-guestbook.yaml](example/guestbook/ing-guestbook.yaml) as the ingress.

This ingress will expose the `frontend` service of the `guestbook-all-in-one` deployment
as a default backend of the Application Gateway.

1. Deploy [ing-guestbook.yaml](example/guestbook/ing-guestbook.yaml) by running

    ```bash
    kubectl apply -f ing-guestbook.yaml
    ```

2. Check the log of the ingress controller for deployment status.

Now the `guestbook` application should be available. You can check this by visiting the
public address of the Application Gateway.

## Deploy Ingress with HTTPS

### Without specified hostname

Without specifying hostname, the guestbook service will be availble on all the hostnames pointing to the application gateway.

1. Before deploying ingress, you need to create a kubernetes secret to host the certificate and private key.
    You can create a kubernetes secret by running

    ```bash
    kubectl create secret tls <guestbook-secret-name> --key <path-to-key> --cert <path-to-cert>
    ```

2. You will be using [ing-guestbook-tls.yaml](example/guestbook/ing-guestbook-tls.yaml) as the ingress. In the ingress,
    specify the name of the secret in the `secretName` section.

    ```yaml
    ...
    spec:
      tls:
        - secretName: <guestbook-secret-name>
    ...
    ```

3. Deploy [ing-guestbook-tls.yaml](example/guestbook/ing-guestbook-tls.yaml) by running

    ```bash
    kubectl apply -f ing-guestbook-tls.yaml
    ```

4. Check the log of the ingress controller for deployment status.

Now the `guestbook` application will be availble on both HTTP and HTTPS.

### With specified hostname

You can also sepcify the hostname on the ingress in order to multiplex TLS configurations and services.
By specifying hostname, the guestbook service will only be availble on the specified host.

1. You will be using [ing-guestbook-tls-sni.yaml](example/guestbook/ing-guestbook-tls-sni.yaml) as the ingress.
    In the ingress, specify the name of the secret in the `secretName` section and replace the hostname accordingly.

    ```yaml
    ...
    spec:
      tls:
        - hosts:
          - <guestbook.contoso.com>
          secretName: <guestbook-secret-name>
      rules:
      - host: <guestbook.contoso.com>
        http:
          paths:
          - backend:
              serviceName: frontend
              servicePort: 80
    ...
    ```

2. Deploy `ing-guestbook-tls-sni.yaml` by running

    ```bash
    kubectl apply -f ing-guestbook-tls-sni.yaml
    ```

3. Check the log of the ingress controller for deployment status.

Now the `guestbook` application will be availble on both HTTP and HTTPS only on the specified host (`<guestbook.contoso.com>` in this example).

## Integrate with other services

You can also add additional paths into this ingress and redirect those paths to other services.
Please take a look at [ing-guestbook-other.yaml](example/guestbook/ing-guestbook-other.yaml).
