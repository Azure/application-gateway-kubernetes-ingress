#!/bin/bash

set -euo pipefail

ENV=${1:-"staging"}
TAG=${2:-$(git describe --abbrev=0 --tags)}

IMAGE_REGISTRY="mcr.microsoft.com/azure-application-gateway/kubernetes-ingress"
CHART_PATH="appgwreg.azurecr.io/public/azure-application-gateway/charts"

echo "Generating Helm chart with tag [$TAG]"
cat ./helm/ingress-azure/Chart-template.yaml | sed "s/XXVERSIONXX/$TAG/g" > ./helm/ingress-azure/Chart.yaml
cat ./helm/ingress-azure/values-template.yaml | sed "s/XXVERSIONXX/$TAG/g" | sed "s#XXREGISTRYXX#$IMAGE_REGISTRY#g" >./helm/ingress-azure/values.yaml
helm package ./helm/ingress-azure --version "$TAG"

CHART_TAR="$(ls -1t ingress-azure-*.tgz | head -n 1)"
echo "Pushing chart $CHART_TAR to $CHART_PATH"

helm push "$CHART_TAR" oci://"$CHART_PATH"
echo "Chart pushed successfully"