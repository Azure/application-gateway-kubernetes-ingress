#!/bin/bash

set -eauo pipefail

TAG=${1:-$(git describe --abbrev=0 --tags)}

OFFICIAL_REGISTRY="mcr.microsoft.com/azure-application-gateway/kubernetes-ingress"
STAGING_REGISTRY="mcr.microsoft.com/azure-application-gateway/kubernetes-ingress-staging"
HELM_GIT_REPO_URL="https://azure.github.io/application-gateway-kubernetes-ingress/helm"
HELM_REPO_URL=${2:-$HELM_GIT_REPO_URL}

echo " - tagging with [$TAG]"
TGZ_FILE=(ingress-azure-$TAG.tgz)

if [ -f $TGZ_FILE ]; then
  echo "File $TGZ_FILE already exists!"
  exit 0
fi

ENV=${3:-""}
REGISTRY=""
if [ "$ENV" = "prod" ]; then
  REGISTRY=$OFFICIAL_REGISTRY  
elif [ "$ENV" = "staging" ]; then
  REGISTRY=$OFFICIAL_REGISTRY
else
  echo " - exiting bad/unknown environment provided: " $ENV
  exit 1
fi

echo " - deployment will use production registry: " $REGISTRY

echo " - update helm templates"
cat ingress-azure/Chart-template.yaml | sed "s/XXVERSIONXX/$TAG/g" > ingress-azure/Chart.yaml
cat ingress-azure/values-template.yaml | sed "s/XXVERSIONXX/$TAG/g" | sed "s#XXREGISTRYXX#$REGISTRY#g" > ingress-azure/values.yaml

echo " - running helm package"
helm package ingress-azure --version "$TAG"

INDEX_FILE_URL="$HELM_REPO_URL/index.yaml"
echo " - check if helm index [$INDEX_FILE_URL] exists in helm repo [$HELM_REPO_URL]"
status_code=$(curl -s --head $INDEX_FILE_URL | head -n 1 | awk '{print $2}')

if [ $status_code -eq "200" ]; then
  echo " - get current helm index from helm repo [$HELM_REPO_URL]"
  curl -s -S $INDEX_FILE_URL -o previous_index.yaml > /dev/null

  echo " - merging with existing helm repo index"
  helm repo index . --url $HELM_REPO_URL --merge previous_index.yaml
else
  echo " - creating a new helm repo index"
  helm repo index . --url $HELM_REPO_URL
fi


echo " - done!"
