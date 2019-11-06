# Tutorials

These tutorials help illustrate the usage of [Kubernetes Ingress Resources](https://kubernetes.io/docs/concepts/services-networking/ingress/) to expose an example Kubernetes service through the [Azure Application Gateway](https://azure.microsoft.com/en-us/services/application-gateway/) over HTTP or HTTPS.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Deploy `guestbook` application](#deploy-guestbook-application)
- [Expose services over HTTP](#expose-services-over-http)
- [Expose services over HTTPS](#expose-services-over-https)
  - [Without specified hostname](#without-specified-hostname)
  - [With specified hostname](#with-specified-hostname)
- [Integrate with other services](#integrate-with-other-services)
- [Automate DNS updates](#automate-dns-updates)

## Prerequisites

- Installed `ingress-azure` helm chart.
  - [**Greenfield Deployment**](setup/install-new.md): If you are starting from scratch, refer to these installation instructions which outlines steps to deploy an AKS cluster with Application Gateway and install application gateway ingress controller on the AKS cluster.
  - [**Brownfield Deployment**](setup/install-existing.md): If you have an existing AKS cluster and Application Gateway, refer to these instructions to install application gateway ingress controller on the AKS cluster.
- If you want to use HTTPS on this application, you will need a x509 certificate and its private key.

## Deploy `guestbook` application

The guestbook application is a canonical Kubernetes application that composes of a Web UI frontend, a backend and a Redis database. By default, `guestbook` exposes its application through a service with name `frontend` on port `80`. Without a Kubernetes Ingress Resource the service is not accessible from outside the AKS cluster. We will use the application and setup Ingress Resources to access the application through HTTP and HTTPS.

Follow the instructions below to deploy the guestbook application.

1. Download `guestbook-all-in-one.yaml` from [here](https://raw.githubusercontent.com/kubernetes/examples/master/guestbook/all-in-one/guestbook-all-in-one.yaml)
1. Deploy `guestbook-all-in-one.yaml` into your AKS cluster by running

  ```bash
  kubectl apply -f guestbook-all-in-one.yaml
  ```

Now, the `guestbook` application has been deployed.

## Expose services over HTTP

In order to expose the guestbook application we will using the following ingress resource:

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: guestbook
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
spec:
  rules:
  - http:
      paths:
      - backend:
          serviceName: frontend
          servicePort: 80
```

This ingress will expose the `frontend` service of the `guestbook-all-in-one` deployment
as a default backend of the Application Gateway.

Save the above ingress resource as `ing-guestbook.yaml`.

1. Deploy `ing-guestbook.yaml` by running:

    ```bash
    kubectl apply -f ing-guestbook.yaml
    ```

1. Check the log of the ingress controller for deployment status.

Now the `guestbook` application should be available. You can check this by visiting the
public address of the Application Gateway.

## Expose services over HTTPS

### Without specified hostname

Without specifying hostname, the guestbook service will be available on all the host-names pointing to the application gateway.

1. Before deploying ingress, you need to create a kubernetes secret to host the certificate and private key. You can create a kubernetes secret by running

    ```bash
    kubectl create secret tls <guestbook-secret-name> --key <path-to-key> --cert <path-to-cert>
    ```

1. Define the following ingress. In the ingress, specify the name of the secret in the `secretName` section.

    ```yaml
    apiVersion: extensions/v1beta1
    kind: Ingress
    metadata:
      name: guestbook
      annotations:
        kubernetes.io/ingress.class: azure/application-gateway
    spec:
      tls:
        - secretName: <guestbook-secret-name>
      rules:
      - http:
          paths:
          - backend:
              serviceName: frontend
              servicePort: 80
    ```

    *NOTE:* Replace `<guestbook-secret-name>` in the above Ingress Resource with the name of your secret. Store the above Ingress Resource in a file name `ing-guestbook-tls.yaml`.

1. Deploy ing-guestbook-tls.yaml by running

    ```bash
    kubectl apply -f ing-guestbook-tls.yaml
    ```

1. Check the log of the ingress controller for deployment status.

Now the `guestbook` application will be available on both HTTP and HTTPS.

### With specified hostname

You can also specify the hostname on the ingress in order to multiplex TLS configurations and services.
By specifying hostname, the guestbook service will only be available on the specified host.

1. Define the following ingress.
    In the ingress, specify the name of the secret in the `secretName` section and replace the hostname in the `hosts` section accordingly.

    ```yaml
    apiVersion: extensions/v1beta1
    kind: Ingress
    metadata:
      name: guestbook
      annotations:
        kubernetes.io/ingress.class: azure/application-gateway
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
    ```

1. Deploy `ing-guestbook-tls-sni.yaml` by running

    ```bash
    kubectl apply -f ing-guestbook-tls-sni.yaml
    ```

1. Check the log of the ingress controller for deployment status.

Now the `guestbook` application will be available on both HTTP and HTTPS only on the specified host (`<guestbook.contoso.com>` in this example).


## Automate DNS updates

When a hostname is specified in the Kubrenetes Ingress resource's rules, it can be used to automaticall create DNS records for the given domain and App Gateway's IP address.
To achieve this the [ExternalDNS](https://github.com/kubernetes-sigs/external-dns) Kubernetes app is required. ExternalDNS in installable via a [Helm chart](https://github.com/kubernetes-incubator/external-dns). The [following document](https://github.com/kubernetes-incubator/external-dns/blob/master/docs/tutorials/azure.md) provides a tutorial on setting up ExternalDNS with an Azure DNS.

Below is a sample Ingress resource, annotated with
`kubernetes.io/ingress.class: azure/application-gateway`, which configures `aplpha.contoso.com`.

```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: websocket-ingress
  namespace: alpha
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
spec:
  rules:
    - host: alpha.contoso.com
      http:
        paths:
        - path: /
          backend:
            serviceName: contoso-service
            servicePort: 80
```

Application Gateway Ingress Controller (AGIC) automatically recognizes the public IP address
assigned to the Application Gateway it is associated with, and sets this IP (`1.2.3.4`)
on the Ingress resource as shown below:

```bash
$ kubectl get ingress -A
NAMESPACE             NAME                HOSTS                 ADDRESS   PORTS   AGE
alpha                 alpha-ingress       alpha.contoso.com     1.2.3.4   80      8m55s
beta                  beta-ingress        beta.contoso.com      1.2.3.4   80      8m54s

```

Once the Ingresses contain both host and adrress, ExternalDNS will provision these to the
DNS system it has been associated with and authorized for.
