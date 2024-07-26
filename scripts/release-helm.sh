#!/bin/bash

set -eauo pipefail

TAG=${1:-$(git describe --abbrev=0 --tags)}
ENV=${2:-"staging"}

echo " - tagging with [$TAG]"
TGZ_FILE=(ingress-azure-$TAG.tgz)

if [ -f $TGZ_FILE ]; then
  echo "File $TGZ_FILE already exists!"
  exit 0
fi

OFFICIAL_REGISTRY="mcr.microsoft.com/azure-application-gateway/kubernetes-ingress"
HELM_OFFICIAL_REPO_URL="https://appgwingress.blob.core.windows.net/ingress-azure-helm-package"
STAGING_REGISTRY="mcr.microsoft.com/azure-application-gateway/kubernetes-ingress-staging"
HELM_STAGING_REPO_URL="https://appgwingress.blob.core.windows.net/ingress-azure-helm-package-staging"
REGISTRY=$STAGING_REGISTRY
HELM_REPO_URL=$HELM_STAGING_REPO_URL
if [ "$ENV" = "prod" ]; then
  REGISTRY=$OFFICIAL_REGISTRY
  HELM_REPO_URL=$HELM_OFFICIAL_REPO_URL
fi

echo " - deployment will use registry: " $REGISTRY

echo " - update helm templates"
cat ./helm/ingress-azure/Chart-template.yaml | sed "s/XXVERSIONXX/$TAG/g" >./helm/ingress-azure/Chart.yaml
cat ./helm/ingress-azure/values-template.yaml | sed "s/XXVERSIONXX/$TAG/g" | sed "s#XXREGISTRYXX#$REGISTRY#g" >./helm/ingress-azure/values.yaml

echo " - running helm package"
helm package ./helm/ingress-azure --version "$TAG"

INDEX_FILE_URL="$HELM_REPO_URL/index.yaml"
echo " - check if helm index [$INDEX_FILE_URL] exists in helm repo [$HELM_REPO_URL]"
status_code=$(curl -s --head $INDEX_FILE_URL | head -n 1 | awk '{print $2}')

if [ $status_code -eq "200" ]; then
  echo " - get current helm index from helm repo [$HELM_REPO_URL]"
  curl -s -S $INDEX_FILE_URL -o previous_index.yaml >/dev/null

  echo " - merging with existing helm repo index"
  helm repo index . --url $HELM_REPO_URL --merge previous_index.yaml
else
  echo " - creating a new helm repo index"
  helm repo index . --url $HELM_REPO_URL
fi

echo " - done!"
