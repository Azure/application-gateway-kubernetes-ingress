#!/bin/bash

set -xeauo pipefail

ENV=${1:-"staging"}
TAG=${2:-$(git describe --abbrev=0 --tags)}

echo "Tagging with [$TAG]"
TAR_FILE="./image/ingress-agic-$TAG.tar:$TAG"

OFFICIAL_REGISTRY="appgwreg.azurecr.io/public/azure-application-gateway/kubernetes-ingress"
STAGING_REGISTRY="appgwreg.azurecr.io/public/azure-application-gateway/kubernetes-ingress-staging"
REGISTRY=$STAGING_REGISTRY
if [ "$ENV" = "prod" ]; then
  REGISTRY=$OFFICIAL_REGISTRY
fi

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Install ORAS if not present
if ! command_exists oras; then
    VERSION="1.1.0"
    echo "Downloading ORAS version $VERSION"
    curl -LO "https://github.com/oras-project/oras/releases/download/v${VERSION}/oras_${VERSION}_linux_amd64.tar.gz"
    mkdir -p oras-install/
    tar -zxf oras_${VERSION}_*.tar.gz -C oras-install/
    sudo mv oras-install/oras /usr/local/bin/
    rm -rf oras_${VERSION}_*.tar.gz oras-install/
    echo "ORAS installed successfully"
else
    echo "ORAS is already installed"
fi

echo "Building multi-arch image"
make build-image-multi-arch

IMAGE_PATH="$REGISTRY:$TAG"
echo "Copying image to $IMAGE_PATH"
oras cp --from-oci-layout $TAR_FILE $IMAGE_PATH

echo "Image pushed successfully"