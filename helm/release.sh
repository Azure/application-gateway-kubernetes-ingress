#!/bin/bash

set -eauo pipefail

TAG=${1:-$(git describe --abbrev=0 --tags)}

echo " - tagging with [$TAG]"
TGZ_FILE=(ingress-azure-$TAG.tgz)

if [ -f $TGZ_FILE ]; then
  echo "File $TGZ_FILE already exists!"
  exit 0
fi

echo " - update helm templates"
cat ingress-azure/Chart-template.yaml | sed "s/XXVERSIONXX/$TAG/g" > ingress-azure/Chart.yaml
cat ingress-azure/values-template.yaml | sed "s/XXVERSIONXX/$TAG/g" > ingress-azure/values.yaml

echo " - running helm package"
helm package ingress-azure --version "$TAG"

echo " - updating helm repo index"
helm repo index . --url https://azure.github.io/application-gateway-kubernetes-ingress/helm


echo " - done!"
