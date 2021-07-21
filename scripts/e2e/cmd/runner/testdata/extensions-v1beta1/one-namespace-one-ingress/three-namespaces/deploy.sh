#!/bin/bash

set -auexo pipefail

echo -e "The goal of this is to ensure that containers with the same probel and same labels in 3 different namespaces have unique and working health probes"

for ns in e2e-ns-x e2e-ns-y e2e-ns-z; do
    kubectl create namespace "${ns}" || true

kubectl apply -f app.yaml