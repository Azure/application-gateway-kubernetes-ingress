#!/bin/bash
set -e
echo " - running helm package"
helm package ingress-azure
echo " - updating helm repo index"
helm repo index . --url https://azure.github.io/application-gateway-kubernetes-ingress/helm
echo " - done!"