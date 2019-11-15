#!/bin/bash

AGIC_NAME="ingress-azure"
AGIC_NAMESPACE="default"
HELM_CONFIG="./helm-config.yaml"

[[ -f $HELM_CONFIG ]] || { echo "File $HELM_CONFIG does not exist!"; exit 1; }


helm template "${AGIC_NAME}" ./helm/ingress-azure \
     --namespace "${AGIC_NAMESPACE=}" \
     --values "${HELM_CONFIG}" \
    | tee /dev/tty | kubectl apply -f -


