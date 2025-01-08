#!/bin/bash

set -euo pipefail

TAG=${1:-$(git describe --abbrev=0 --tags)}
ENV=${2:-"staging"}

IMAGE_REGISTRY="mcr.microsoft.com/azure-application-gateway/kubernetes-ingress"
CHART_PATH="appgwreg.azurecr.io/public/azure-application-gateway/charts"

echo " - tagging with [$TAG]"

echo " - update helm templates"
cat ./helm/ingress-azure/Chart-template.yaml | sed "s/XXVERSIONXX/$TAG/g" > ./helm/ingress-azure/Chart.yaml
cat ./helm/ingress-azure/values-template.yaml | sed "s/XXVERSIONXX/$TAG/g" | sed "s#XXREGISTRYXX#$IMAGE_REGISTRY#g" >./helm/ingress-azure/values.yaml
helm package ./helm/ingress-azure --version "$TAG"

CHART_TAR="$(ls -1t ingress-azure-*.tgz | head -n 1)"
echo " - pushing chart $CHART_TAR to $CHART_PATH"

helm push "$CHART_TAR" oci://"$CHART_PATH"