#!/bin/bash

set -auexo pipefail

echo -e "The goal of this is to ensure health probe is generated from container readiness probe and backend should be removed when the probe is unhealthy"

for ns in e2e-probe1 e2e-probe2; do
    kubectl create namespace "${ns}" || true

kubectl apply -f app.yaml