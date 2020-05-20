#!/bin/bash

set -auexo pipefail

kubectl create namespace e2e-three-ings || true

cat <<EOF | KUBECONFIG=$HOME/.kube/config kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: httpbin
  namespace: e2e-three-ings
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

apiVersion: apps/v1
kind: Deployment
metadata:
  name: aspnetapp
  namespace: e2e-three-ings
spec:
  selector:
    matchLabels:
      app: aspnetapp
  replicas: 1
  template:
    metadata:
      labels:
        app: aspnetapp
    spec:
      containers:
        - name: apsnetapp
          imagePullPolicy: Always
          image: mcr.microsoft.com/dotnet/core/samples:aspnetapp
          ports:
            - containerPort: 80

---

apiVersion: apps/v1
kind: Deployment
metadata:
  name: kuard
  namespace: e2e-three-ings
spec:
  selector:
    matchLabels:
      app: kuard
  replicas: 1
  template:
    metadata:
      labels:
        app: kuard
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
  namespace: e2e-three-ings
spec:
  selector:
    app: httpbin
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80

---

apiVersion: v1
kind: Service
metadata:
  name: aspnetapp
  namespace: e2e-three-ings
spec:
  selector:
    app: aspnetapp
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80

---

apiVersion: v1
kind: Service
metadata:
  name: kuard
  namespace: e2e-three-ings
spec:
  selector:
    app: kuard
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80

---

apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: ingress-a
  namespace: e2e-three-ings
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/backend-path-prefix: "/"
spec:
  rules:
    - host: ws.mis.li
      http:
        paths:
        - path: /*
          backend:
            serviceName: aspnetapp
            servicePort: 80

---

apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: ingress-b
  namespace: e2e-three-ings
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/backend-path-prefix: "/"
spec:
  rules:
    - host: ws.mis.li
      http:
        paths:
        - path: /igloo
          backend:
            serviceName: httpbin
            servicePort: 80

---

apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: ingress-c
  namespace: e2e-three-ings
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/backend-path-prefix: "/"
spec:
  rules:
    - host: ws.mis.li
      http:
        paths:
        - path: /kuard
          backend:
            serviceName: kuard
            servicePort: 80

---

apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: ingress-d
  namespace: e2e-three-ings
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/backend-path-prefix: "/"
spec:
  backend:
    serviceName: kuard
    servicePort: 80
EOF
