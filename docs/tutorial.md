# Table of Contents
- [Prerequisites](#prerequisites)
- [Deploy `guestbook` application](#deploy-guestbook-application)
- [Expose services over HTTP](#expose-services-over-http)
- [Expose services over HTTPS](#expose-services-over-https)
  * [Without specified hostname](#without-specified-hostname)
  * [With specified hostname](#with-specified-hostname)
  * [Certificate issuance with `Lets Encrypt`](#certificate-issuance-with-lets-encrypt)
- [Integrate with other services](#integrate-with-other-services)
- [Adding Health Probes to your service](#adding-health-probes-to-your-service)
  * [With readinessProbe or livenessProbe](#with-readinessprobe-or-livenessprobe)
  * [Without readinessProbe or livenessProbe](#without-readinessprobe-or-livenessprobe)
  * [Default Values for Health Probe](#default-values-for-health-probe)
- [Enable Cookie Based Affinity](#enable-cookie-based-affinity)
- [Expose a WebSocket server](#expose-a-websocket-server)

# Tutorials
These tutorials help illustrate the usage of [Kubernetes Ingress Resources](https://kubernetes.io/docs/concepts/services-networking/ingress/) to expose an example Kubernetes service through the [Azure Application Gateway](https://azure.microsoft.com/en-us/services/application-gateway/) over HTTP or HTTPS.

## Prerequisites

- Installed `ingress-azure` helm chart.
    - **Greenfield Deployment**: If you are starting from scratch, refer to these [installation](install-new.md) instructions which outlines steps to deploy an AKS cluster with Application Gateway and install application gateway ingress controller on the AKS cluster.
    - **Brownfield Deployment**: If you have an existing AKS cluster and Application Gateway, refer to these [installation](install-existing.md) instructions to install application gateway ingress controller on the AKS cluster.
- If you want to use HTTPS on this application, you will need a x509 certificate and its private key.

## Deploy `guestbook` application

The guestbook application is a canonical Kubernetes application that composes of a Web UI frontend, a backend and a Redis database. By default, `guestbook` exposes its application through a service with name `frontend` on port `80`. Without a Kubernetes Ingress Resource the service is not accessible from outside the AKS cluster. We will use the application and setup Ingress Resources to access the application through HTTP and HTTPS.

Follow the instructions below to deploy the guestbook application.

1. Download `guestbook-all-in-one.yaml` from [here](https://raw.githubusercontent.com/kubernetes/examples/master/guestbook/all-in-one/guestbook-all-in-one.yaml)
2. Deploy `guestbook-all-in-one.yaml` into your AKS cluster by running

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

1. Deploy `ing-guestbook.yaml` by running

    ```bash
    kubectl apply -f ing-guestbook.yaml
    ```

2. Check the log of the ingress controller for deployment status.

Now the `guestbook` application should be available. You can check this by visiting the
public address of the Application Gateway.

## Expose services over HTTPS

### Without specified hostname

Without specifying hostname, the guestbook service will be available on all the host-names pointing to the application gateway.

1. Before deploying ingress, you need to create a kubernetes secret to host the certificate and private key.
    You can create a kubernetes secret by running

    ```bash
    kubectl create secret tls <guestbook-secret-name> --key <path-to-key> --cert <path-to-cert>
    ```

2. Define the following ingress. In the ingress, specify the name of the secret in the `secretName` section.

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
    *NOTE:* Replace `<guestbook-secret-name>` in the above Ingress Resource with the name of your secret.

      Store the above Ingress Resource in a file name `ing-guestbook-tls.yaml`.

3. Deploy ing-guestbook-tls.yaml by running

    ```bash
    kubectl apply -f ing-guestbook-tls.yaml
    ```

4. Check the log of the ingress controller for deployment status.

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

2. Deploy `ing-guestbook-tls-sni.yaml` by running

    ```bash
    kubectl apply -f ing-guestbook-tls-sni.yaml
    ```

3. Check the log of the ingress controller for deployment status.

Now the `guestbook` application will be available on both HTTP and HTTPS only on the specified host (`<guestbook.contoso.com>` in this example).

### Certificate issuance with LetsEncrypt.org

This section configures your AKS to leverage [LetsEncrypt.org](https://letsencrypt.org/) and automatically obtain a
TLS/SSL certificate for your domain. The certificate will be installed on Application Gateway, which will perform
SSL/TLS termination for your AKS cluster. The setup described here uses the
[cert-manager](https://github.com/jetstack/cert-manager) Kubernetes add-on, which automates the creation and management of 
certificates.

Follow the steps below to install [cert-manager](https://docs.cert-manager.io) on your existing AKS cluster.

##### 1. Helm Chart

Run the following script to install the `cert-manager` helm chart. This will:
  - create a new `cert-manager` namespace on your AKS
  - create the following CRDs: Certificate, Challenge, ClusterIssuer, Issuer, Order
  - install cert-manager chart (from [docs.cert-manager.io)](https://docs.cert-manager.io/en/latest/getting-started/install/kubernetes.html#steps)


```bash
#!/bin/bash

# Install the CustomResourceDefinition resources separately
kubectl apply -f https://raw.githubusercontent.com/jetstack/cert-manager/release-0.8/deploy/manifests/00-crds.yaml

# Create the namespace for cert-manager
kubectl create namespace cert-manager

# Label the cert-manager namespace to disable resource validation
kubectl label namespace cert-manager certmanager.k8s.io/disable-validation=true

# Add the Jetstack Helm repository
helm repo add jetstack https://charts.jetstack.io

# Update your local Helm chart repository cache
helm repo update

# Install the cert-manager Helm chart
helm install \
  --name cert-manager \
  --namespace cert-manager \
  --version v0.8.0 \
  jetstack/cert-manager
```

##### 2. ClusterIssuer Resource

Create a `ClusterIssuer` resource. It is required by `cert-manager` to represent the `Lets Encrypt` certificate
authority where the signed certificates will be obtained.

By using the non-namespaced `ClusterIssuer` resource, cert-manager will issue certificates that can be consumed from
multiple namespaces. `Let’s Encrypt` uses the ACME protocol to verify that you control a given domain name and to issue
you a certificate. More details on configuring `ClusterIssuer` properties
[here](https://docs.cert-manager.io/en/latest/tasks/issuers/index.html). `ClusterIssuer` will instruct `cert-manager`
to issue certificates using the `Lets Encrypt` staging environment used for testing (the root certificate not present
in browser/client trust stores).

The default challenge type in the YAML below is `http01`. Other challenges are documented on [letsencrypt.org - Challenge Types](https://letsencrypt.org/docs/challenge-types/)

**IMPORTANT:** Update `<YOUR.EMAIL@ADDRESS>` in the YAML below

```bash
#!/bin/bash
kubectl apply -f - <<EOF
apiVersion: certmanager.k8s.io/v1alpha1
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
spec:
  acme:
    # You must replace this email address with your own.
    # Let's Encrypt will use this to contact you about expiring
    # certificates, and issues related to your account.
    email: <YOUR.EMAIL@ADDRESS>
    # ACME server URL for Let’s Encrypt’s staging environment.
    # The staging environment will not issue trusted certificates but is
    # used to ensure that the verification process is working properly
    # before moving to production
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    privateKeySecretRef:
      # Secret resource used to store the account's private key.
      name: example-issuer-account-key
    # Enable the HTTP-01 challenge provider
    # you prove ownership of a domain by ensuring that a particular
    # file is present at the domain
    http01: {}
EOF
```

##### 3. Deploy App

Create an Ingress resource to Expose the `guestbook` application using the Application Gateway with the Lets Encrypt Certificate.

Ensure you Application Gateway has a public Frontend IP configuration with a DNS name (either using the
default `azure.com` domain, or provision a `Azure DNS Zone` service, and assign your own custom domain).
Note the annotation `certmanager.k8s.io/cluster-issuer: letsencrypt-staging`, which tells cert-manager to process the
tagged Ingress resource.

**IMPORTANT:**  Update `<PLACEHOLDERS.COM>` in the YAML below with your own domain (or the Application Gateway one, for example
'kh-aks-ingress.westeurope.cloudapp.azure.com')

```bash
kubectl apply -f - <<EOF
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: guestbook-letsencrypt-staging
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    certmanager.k8s.io/cluster-issuer: letsencrypt-staging
spec:
  tls:
  - hosts:
    - <PLACEHOLDERS.COM>
    secretName: guestbook-secret-name
  rules:
  - host: <PLACEHOLDERS.COM>
    http:
      paths:
      - backend:
          serviceName: frontend
          servicePort: 80
EOF
```

After a few seconds, you  can access the `guestbook` service through the Application Gateway HTTPS url using the automatically issued **staging** `Lets Encrypt` certificate.
Your browser may warn you of an invalid cert authority. The staging certificate is issued by `CN=Fake LE Intermediate X1`. This is an indication that the system worked as expected and you are ready for your production certificate.


##### 4. Production Certificate
Once your staging certificate is setup successfully you can switch to a production ACME server:
  1. Replace the staging annotation on your Ingress resource with: `certmanager.k8s.io/cluster-issuer: letsencrypt-prod`
  2. Delete the existing staging `ClusterIssuer` you created in the previous step and create a new one by replacing the ACME server from the ClusterIssuer YAML above with `https://acme-v02.api.letsencrypt.org/directory`

##### 5. Certificate Expiration and Renewal
Before the `Lets Encrypt` certificate expires, `cert-manager` will automatically update the certificate in the Kubernetes secret store. At that point, Application Gateway Ingress Controller will apply the updated secret referenced in the ingress resources it is using to configure the Application Gateway.

## Integrate with other services

The following ingress will allow you to add additional paths into this ingress and redirect those paths to other services:

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
      - path: </other/*>
        backend:
          serviceName: <other-service>
          servicePort: 80
      - backend:
          serviceName: frontend
          servicePort: 80
```

## Adding Health Probes to your service
By default, Ingress controller will provision an HTTP GET probe for the exposed pods.
The probe properties can be customized by adding a [Readiness or Liveness Probe](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-probes/) to your `deployment`/`pod` spec.

### With `readinessProbe` or `livenessProbe`
```yaml
apiVersion: extensions/v1beta1
kind: Deployment
metadata:
  name: aspnetapp
spec:
  replicas: 3
  template:
    metadata:
      labels:
        service: site
    spec:
      containers:
      - name: aspnetapp
        image: mcr.microsoft.com/dotnet/core/samples:aspnetapp
        imagePullPolicy: IfNotPresent
        ports:
        - containerPort: 80
        readinessProbe:
          httpGet:
            path: /
            port: 80
          periodSeconds: 3
          timeoutSeconds: 1
```

Kubernetes API Reference:
* [Container Probes](https://kubernetes.io/docs/concepts/workloads/pods/pod-lifecycle/#container-probes)
* [HttpGet Action](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.14/#httpgetaction-v1-core)

*Note:*
1) `readinessProbe` and `livenessProbe` are supported when configured with `httpGet`.
2) Probing on a port other than the one exposed on the pod is currently not supported.
3) `HttpHeaders`, `InitialDelaySeconds`, `SuccessThreshold` are not supported.

###  Without `readinessProbe` or `livenessProbe`
If the above probes are not provided, then Ingress Controller make an assumption that the service is reachable on `Path` specified for `backend-path-prefix` annotation or the `path` specified in the `ingress` definition for the service.

### Default Values for Health Probe
For any property that can not be inferred by the readiness/liveness probe, Default values are set.

| Application Gateway Probe Property | Default Value |
|-|-|
| `Path` | / |
| `Host` | localhost |
| `Protocol` | HTTP |
| `Timeout` | 30 |
| `Interval` | 30 |
| `UnhealthyThreshold` | 3 |

## Enable Cookie based Affinity
As outlined in the [Azure Application Gateway Documentation](https://docs.microsoft.com/en-us/azure/application-gateway/application-gateway-components#http-settings), Application Gateway supports cookie based affinity enabling which it can direct subsequent traffic from a user session to the same server for processing.

### Example
```yaml
apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: guestbook
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/cookie-based-affinity: "true"
spec:
  rules:
  - http:
      paths:
      - backend:
          serviceName: frontend
          servicePort: 80
```

## Expose a WebSocket server

As outlined in the Application Gateway v2 documentation - it [provides native support for the WebSocket and HTTP/2 protocols](https://docs.microsoft.com/en-us/azure/application-gateway/overview#websocket-and-http2-traffic). Please note, that for both Application Gateway and the Kubernetes Ingress - there is no user-configurable setting to selectively enable or disable WebSocket support.

The Kubernetes deployment YAML below shows the minimum configuration used to deploy a WebSocket server, which is the same as deploying a regular web server:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: websocket-server
spec:
  selector:
    matchLabels:
      app: ws-app
  replicas: 2
  template:
    metadata:
      labels:
        app: ws-app
    spec:
      containers:
        - name: websocket-app
          imagePullPolicy: Always
          image: your-container-repo.azurecr.io/websockets-app
          ports:
            - containerPort: 8888
      imagePullSecrets:
        - name: azure-container-registry-credentials

---

apiVersion: v1
kind: Service
metadata:
  name: websocket-app-service
spec:
  selector:
    app: ws-app
  ports:
  - protocol: TCP
    port: 80
    targetPort: 8888

---

apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: websocket-repeater
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
spec:
  rules:
    - host: ws.contoso.com
      http:
        paths:
          - backend:
              serviceName: websocket-app-service
              servicePort: 80
```

Given that all the prerequisites are fulfilled, and you have an App Gateway controlled by a K8s Ingress in your AKS, the deployment above would result in a WebSockets server exposed on port 80 of your App Gateway's public IP and the `ws.contoso.com` domain.

The following cURL command would test the WebSocket server deployment:
```sh
curl -i -N -H "Connection: Upgrade" \
        -H "Upgrade: websocket" \
        -H "Origin: http://localhost" \
        -H "Host: ws.contoso.com" \
        -H "Sec-Websocket-Version: 13" \
        -H "Sec-WebSocket-Key: 123" \
        http://1.2.3.4:80/ws
```

##### WebSocket Health Probes

If your deployment does not explicitly define health probes, App Gateway would attempt an  HTTP GET on your WebSocket server endpoint.
Depending on the server implementation ([here is one we love](https://github.com/gorilla/websocket/blob/master/examples/chat/main.go)) WebSocket specific headers may be required (`Sec-Websocket-Version` for instance).
Since App Gateway does not add WebSocket headers, the App Gateway's health probe response from your WebSocket server will most likely be `400 Bad Request`.
As a result App Gateway will mark your pods as unhealthy, which will eventually result in a `502 Bad Gateway` for the consumers of the WebSocket server.
To avoid this you may need to add an HTTP GET handler for a health check to your server (`/health` for instance, which returns `200 OK`).
