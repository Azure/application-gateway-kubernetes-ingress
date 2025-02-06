# Tutorial: Setting up E2E SSL

> **_NOTE:_** [Application Gateway for Containers](https://aka.ms/agc) has been released, which introduces numerous performance, resilience, and feature changes. Please consider leveraging Application Gateway for Containers for your next deployment.

In this this tutorial, we will learn how to setup E2E SSL with AGIC on Application Gateway.

We will

1. Generate the frontend and the backend certificates
1. Deploy a simple application with HTTPS
1. Upload the backend certificate's root certificate to Application Gateway
1. Setup ingress for E2E

> **Note: Following tutorial makes use of test certificate generated using OpenSSL. These certificates are only for illustration and should be used in testing only.**

## Generate the frontend and the backend certificates

Let's start by first generating the certificates that we will be using for the frontend and backend SSL.

1. First, we will generate the frontend certificate that will be presented to the clients connecting to the Application Gateway. This will have subject name `CN=frontend`.

    ```bash
    openssl ecparam -out frontend.key -name prime256v1 -genkey
    openssl req -new -sha256 -key frontend.key -out frontend.csr -subj "/CN=frontend"
    openssl x509 -req -sha256 -days 365 -in frontend.csr -signkey frontend.key -out frontend.crt
    ```

    > Note: You can also use a [certificate present on the Key Vault](../features/appgw-ssl-certificate.md) on Application Gateway for frontend SSL.

1. Now, we will generate the backend certificate that will be presented by the backends to the Application Gateway. This will have subject name `CN=backend`

    ```bash
    openssl ecparam -out backend.key -name prime256v1 -genkey
    openssl req -new -sha256 -key backend.key -out backend.csr -subj "/CN=backend"
    openssl x509 -req -sha256 -days 365 -in backend.csr -signkey backend.key -out backend.crt
    ```

1. Finally, we will install the above certificates on to our kubernetes cluster

    ```bash
    kubectl create secret tls frontend-tls --key="frontend.key" --cert="frontend.crt"
    kubectl create secret tls backend-tls --key="backend.key" --cert="backend.crt"
    ```

    Here is output after listing the secrets.

    ```bash
    > kubectl get secrets
    NAME                  TYPE                                  DATA   AGE
    backend-tls           kubernetes.io/tls                     2      3m18s
    frontend-tls          kubernetes.io/tls                     2      3m18s
    ```

## Deploy a simple application with HTTPS

In this section, we will deploy a simple application exposing an HTTPS endpoint on port 8443.

```yaml
apiVersion: v1
kind: Service
metadata:
  name: website-service
spec:
  selector:
    app: website
  ports:
  - protocol: TCP
    port: 8443
    targetPort: 8443
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: website-deployment
spec:
  selector:
    matchLabels:
      app: website
  replicas: 2
  template:
    metadata:
      labels:
        app: website
    spec:
      containers:
        - name: website
          imagePullPolicy: Always
          image: nginx:latest
          ports:
            - containerPort: 8443
          volumeMounts:
          - mountPath: /etc/nginx/ssl
            name: secret-volume
          - mountPath: /etc/nginx/conf.d
            name: configmap-volume
      volumes:
      - name: secret-volume
        secret:
          secretName: backend-tls
      - name: configmap-volume
        configMap:
          name: website-nginx-cm
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: website-nginx-cm
data:
  default.conf: |-
    server {
        listen 8080 default_server;
        listen 8443 ssl;
        root /usr/share/nginx/html;
        index index.html;
        ssl_certificate /etc/nginx/ssl/tls.crt;
        ssl_certificate_key /etc/nginx/ssl/tls.key;
        location / {
          return 200 "Hello World!";
        }
    }
```

You can also install the above yamls using:

```bash
kubectl apply -f https://raw.githubusercontent.com/Azure/application-gateway-kubernetes-ingress/master/docs/examples/sample-https-backend.yaml
```

Verify that you can curl the application

```bash
> kubectl get pods
NAME                                 READY   STATUS    RESTARTS   AGE
website-deployment-9c8c6df7f-5bqwh   1/1     Running   0          24s
website-deployment-9c8c6df7f-wxtnp   1/1     Running   0          24s

> kubectl exec -it website-deployment-9c8c6df7f-5bqwh -- curl -k https://localhost:8443
Hello World!
```

## Upload the backend certificate's root certificate to Application Gateway

When you are setting up SSL between Application Gateway and Backend, if you are using a self-signed certificate or a certificate signed by a custom root CA on the backend, then you need to upload self-signed or the Custom root CA of the backend certificate on the Application Gateway.

```bash
applicationGatewayName="<gateway-name>"
resourceGroup="<resource-group>"
az network application-gateway root-cert create \
    --gateway-name $applicationGatewayName  \
    --resource-group $resourceGroup \
    --name backend-tls \
    --cert-file backend.crt
```

## Setup ingress for E2E

Now, we will configure our ingress to use the `frontend` certificate for frontend SSL and `backend` certificate as root certificate so that Application Gateway can authenticate the backend.

```bash
cat << EOF | kubectl apply -f -
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: website-ingress
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/ssl-redirect: "true"
    appgw.ingress.kubernetes.io/backend-protocol: "https"
    appgw.ingress.kubernetes.io/backend-hostname: "backend"
    appgw.ingress.kubernetes.io/appgw-trusted-root-certificate: "backend-tls"
spec:
  tls:
    - secretName: frontend-tls
      hosts:
        - website.com
  rules:
    - host: website.com
      http:
        paths:
        - path: /
          backend:
            service:
              name: website-service
              port:
                number: 8443
          pathType: Exact
EOF
```

For frontend SSL, we have added `tls` section in our ingress resource.

```yaml
  tls:
    - secretName: frontend-tls
      hosts:
        - website.com
```

For backend SSL, we have added the following annotations:

```yaml
appgw.ingress.kubernetes.io/backend-protocol: "https"
appgw.ingress.kubernetes.io/backend-hostname: "backend"
appgw.ingress.kubernetes.io/appgw-trusted-root-certificate: "backend-tls"
```

Here, it is important to note that `backend-hostname` should be the hostname that the backend will accept and it should also match with the Subject/Subject Alternate Name of the certificate used on the backend.

After you have successfully completed all the above steps, you should be able to see the ingress's IP address and visit the website.

```bash
> kubectl get ingress
NAME              HOSTS         ADDRESS         PORTS     AGE
website-ingress   website.com   <gateway-ip>   80, 443   36m

> curl -k -H "Host: website.com" https://<gateway-ip>
Hello World!
```
