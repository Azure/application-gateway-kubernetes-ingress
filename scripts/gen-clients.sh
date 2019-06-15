#!/bin/bash

set -auexo pipefail

echo -e "Cleanup previously generated code..."
rm -rf pkg/client $(find ./pkg -name 'zz_*.go')

echo -e "Generate All..."
../code-generator/generate-groups.sh \
    all \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/client \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis \
    "azureingressmanagedtarget:v1 azureingressprohibitedtarget:v1"
