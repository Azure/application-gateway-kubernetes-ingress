#!/bin/bash

set -auexo pipefail

kubectl create namespace e2e-three-ings || true

kubectl apply -f app.yaml