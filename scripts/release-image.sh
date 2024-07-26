#!/bin/bash

set -xeauo pipefail

TAG=${1:-$(git describe --abbrev=0 --tags)}
ENV=${2:-"staging"}

echo " - tagging with [$TAG]"
TAR_FILE=("./image/ingress-agic-$TAG.tar:$TAG")

OFFICIAL_REGISTRY="appgwreg.azurecr.io/public/azure-application-gateway/kubernetes-ingress"
STAGING_REGISTRY="appgwreg.azurecr.io/public/azure-application-gateway/kubernetes-ingress-staging"
REGISTRY=$STAGING_REGISTRY
if [ "$ENV" = "prod" ]; then
  REGISTRY=$OFFICIAL_REGISTRY
fi

VERSION="1.1.0"
curl -LO "https://github.com/oras-project/oras/releases/download/v${VERSION}/oras_${VERSION}_linux_amd64.tar.gz"
mkdir -p oras-install/
tar -zxf oras_${VERSION}_*.tar.gz -C oras-install/
sudo mv oras-install/oras /usr/local/bin/
rm -rf oras_${VERSION}_*.tar.gz oras-install/

IMAGE_PATH="$REGISTRY:$TAG"
oras cp --from-oci-layout $TAR_FILE $IMAGE_PATH
