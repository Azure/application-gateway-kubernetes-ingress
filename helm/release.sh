#!/bin/bash

set -eauo pipefail

GIT_TAG=$(git describe --abbrev=0 --tags)

TGZ_FILE="ingress-azure-$GIT_TAG.tgz"

if [ -f "$TGZ_FILE" ]; then
  echo "File $TGZ_FILE already exists!"
  exit 0
fi

echo " - update helm templates"
sed "s/XXVERSIONXX/$GIT_TAG/g" ingress-azure/Chart-template.yaml > ingress-azure/Chart.yaml
sed "s/XXVERSIONXX/$GIT_TAG/g" ingress-azure/values-template.yaml > ingress-azure/values.yaml

echo " - running helm package"
helm package ingress-azure --version "$GIT_TAG"

echo " - updating helm repo index"
helm repo index . --url https://azure.github.io/application-gateway-kubernetes-ingress/helm

echo " - done!"
