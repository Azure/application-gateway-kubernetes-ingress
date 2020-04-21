#!/bin/bash
exit 0

set -auexo pipefail

# Install the CustomResourceDefinition resources separately
# kubectl apply -f https://raw.githubusercontent.com/jetstack/cert-manager/release-0.8/deploy/manifests/00-crds.yaml
#kubectl apply -f https://raw.githubusercontent.com/jetstack/cert-manager/release-0.11/deploy/manifests/00-crds.yaml

kubectl apply \
    -f https://raw.githubusercontent.com/jetstack/cert-manager/release-0.11/deploy/manifests/00-crds.yaml \
    --validate=false

# Create the namespace for cert-manager
kubectl create namespace cert-manager  || true

# Label the cert-manager namespace to disable resource validation
kubectl label namespace cert-manager certmanager.k8s.io/disable-validation=true || true

# Add the Jetstack Helm repository
helm repo add jetstack https://charts.jetstack.io  || true

# Update your local Helm chart repository cache
helm repo update  || true

# Install the cert-manager Helm chart
helm install \
  --name cert-manager \
  --namespace cert-manager \
  --version v0.11.0 \
  jetstack/cert-manager  || true


kubectl create namespace lencr || true

kubectl apply -f secret.yaml || true

kubectl apply -f - <<EOF
apiVersion: cert-manager.io/v1alpha2
kind: ClusterIssuer
metadata:
  name: letsencrypt-staging
spec:
  acme:

    # You must replace this email address with your own.
    # Let's Encrypt will use this to contact you about expiring
    # certificates, and issues related to your account.
    email: delyan.raychev@microsoft.com

    # ACME server URL for Let’s Encrypt’s staging environment.
    # The staging environment will not issue trusted certificates but is
    # used to ensure that the verification process is working properly
    # before moving to production
    server: https://acme-staging-v02.api.letsencrypt.org/directory

    privateKeySecretRef:
      # Secret resource used to store the account's private key.
      name: letsencrypt-secret

    # Enable the HTTP-01 challenge provider
    # you prove ownership of a domain by ensuring that a particular
    # file is present at the domain
    solvers:
    - http01:
        ingress:
            class: azure/application-gateway
EOF


kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin
  namespace: lencr
spec:
  selector:
    matchLabels:
      app: httpbin
  replicas: 1
  template:
    metadata:
      labels:
        app: httpbin
    spec:
      containers:
        - name: httpbin
          imagePullPolicy: Always
          image: docker.io/kennethreitz/httpbin
          ports:
            - containerPort: 80

---

apiVersion: v1
kind: Service
metadata:
  name: httpbin
  namespace: lencr
spec:
  selector:
    app: httpbin
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80

---

apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: ingress-b
  namespace: lencr
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/backend-path-prefix: "/"
    appgw.ingress.kubernetes.io/ssl-redirect: "true"
    cert-manager.io/cluster-issuer: letsencrypt-staging
    cert-manager.io/acme-challenge-type: http01
spec:
  tls:
   - hosts:
     - www.ghettote.ch
     secretName: ghettotech-secret
  rules:
    - host: www.ghettote.ch
      http:
        paths:
        - path: /
          backend:
            serviceName: httpbin
            servicePort: 80
EOF

kubectl delete namespace/cert-manager