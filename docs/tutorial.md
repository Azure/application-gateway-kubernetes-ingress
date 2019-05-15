- [Prerequisites](#prerequisites)
- [Deploy `guestbook` application](#deploy-guestbook-application)
- [Expose services over HTTP](#expose-services-over-http)
- [Expose services over HTTPS](#expose-services-over-https)
  * [Without specified hostname](#without-specified-hostname)
  * [With specified hostname](#with-specified-hostname)
- [Integrate with other services](#integrate-with-other-services)
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


## Expose a WebSocket server

As outlined in the Application Gateway v2 documentation - it [provides native support for the WebSocket and HTTP/2 protocols](https://docs.microsoft.com/en-us/azure/application-gateway/overview#websocket-and-http2-traffic). For both App Gateway and the Kubernetes Ingress - there is no user-configurable setting to selectively enable or disable WebSocket support.

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
