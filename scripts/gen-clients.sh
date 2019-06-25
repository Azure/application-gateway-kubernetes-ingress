#!/bin/bash

set -auexo pipefail

# This script requres a checkout of https://github.com/kubernetes/code-generator in ../
# For more information read https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/

echo -e "Cleanup previously generated code..."
rm -rf pkg/client $(find ./pkg -name 'zz_*.go')

echo -e "Generate AzureIngressManagedTarget, AzureIngressProhibitedTarget..."
../code-generator/generate-groups.sh \
    all \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/client \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis \
    "azureingressmanagedtarget:v1 azureingressprohibitedtarget:v1"

go get github.com/knative/pkg/apis/istio/v1alpha3

go vendor

echo -e "Generate Istio CRDs..."
../code-generator/generate-groups.sh \
    all \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/istio_client \
    github.com/knative/pkg/apis "istio:v1alpha3"
