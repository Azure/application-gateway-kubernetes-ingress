#!/bin/bash

set -auexo pipefail

echo -e "The goal of this is to ensure that containers with the same probel and same labels in 3 different namespaces have unique and working health probes"

for ns in ns-x ns-y ns-z; do
    kubectl create namespace "${ns}" || true

kubectl apply -f - <<EOF
apiVersion: apps/v1
kind: Deployment
metadata:
  name: zagora-deployment
  namespace: ${ns}
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
        - name: zagora-app
          imagePullPolicy: Always
          image: docker.io/kennethreitz/httpbin
          ports:
            - containerPort: 80
          livenessProbe:
            httpGet:
              path: /status/200
              port: 80
            initialDelaySeconds: 3
            periodSeconds: 3
      imagePullSecrets:
        - name: acr-creds

---

apiVersion: v1
kind: Service
metadata:
  name: zagora-service
  namespace: ${ns}
spec:
  selector:
    app: ws-app
  ports:
  - protocol: TCP
    port: 80
    targetPort: 80

---

apiVersion: extensions/v1beta1
kind: Ingress
metadata:
  name: zagora-ingress
  namespace: ${ns}
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
spec:
  rules:
    - host: ws-${ns}.mis.li
      http:
        paths:
        - path: /*
          backend:
            serviceName: zagora-service
            servicePort: 80
EOF
done
