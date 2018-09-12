# Example: Guestbook

This example will show you how to deploy a multi-tier web application and use a
simple ingress to configure the application gateway

## Prerequisite

- Installed `application-gateway-kubernetes-ingress` helm chart (see [here](install))

## Deploy `guestbook`

1. Download `guestbook-all-in-one.yaml` from [here](https://github.com/kubernetes/examples/blob/master/guestbook/all-in-one/guestbook-all-in-one.yaml)
2. Deploy `guestbook-all-in-one.yaml` into your AKS cluster by running

    ```bash
    kubectl apply -f guestbook-all-in-one.yaml
    ```

## Deploy ingress

Now, the `guestbook` application has been deployed.

By default, it exposes its frontend through a service with name `frontend` on port `80`.

We will be using [ing-frontend.yaml](example/guestbook/ing-guestbook.yaml) as the ingress.

This ingress will expose the `frontend` service of the `guestbook-all-in-one` deployment
as a default backend of the Application Gateway.

1. Deploy `ing-frontend.yaml` by running

    ```bash
    kubectl apply -f ing-frontend.yaml
    ```

2. Check the log of the ingress controller for deployment status.

Now the `guestbook` application should be available. You can check this by visiting the
public address of the Application Gateway.

You can also add additional paths into this ingress and redirect those paths to other services.
Please take a look at [ing-frontend.yaml](example/guestbook/ing-guestbook-other.yaml)