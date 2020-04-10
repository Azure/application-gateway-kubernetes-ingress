#!/bin/bash

set -auexo pipefail

# This script requres a checkout of https://github.com/kubernetes/code-generator in ../
# To be backward compatible with an old version of client-go(v11.0.0), copy vendor/k8s.io/code-generator in ../ to use instead checkout as above

# For more information read https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/

echo -e "Cleanup previously generated code..."
rm -rf pkg/client $(find ./pkg -name 'zz_*.go')

echo -e "Generate AzureBackendPool CRD..."
../code-generator/generate-groups.sh \
    all \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/client \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis \
    "azurebackendpool:v1" \
    --go-header-file ../code-generator/hack/boilerplate.go.txt

echo -e "Generate AzureIngressProhibitedTarget CRD..."
../code-generator/generate-groups.sh \
    all \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/client \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis \
    "azureingressprohibitedtarget:v1" \
    --go-header-file ../code-generator/hack/boilerplate.go.txt    

go get github.com/knative/pkg/apis/istio/v1alpha3

go mod vendor

echo -e "Generate Istio CRDs..."
../code-generator/generate-groups.sh \
    all \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/istio_client \
    github.com/knative/pkg/apis "istio:v1alpha3" \
    --go-header-file ../code-generator/hack/boilerplate.go.txt