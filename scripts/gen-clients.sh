#!/bin/bash

set -auexo pipefail

# This script requres a checkout of https://github.com/kubernetes/code-generator in ../
# For more information read https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/

# echo -e "Cleanup previously generated code..."
# rm -rf pkg/client $(find ./pkg -name 'zz_*.go')

echo -e "Generate CRD..."
../code-generator/generate-groups.sh \
    all \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis \
    "azureapplicationgatewayinstanceupdatestatus:v1beta1 azureapplicationgatewaybackendpool:v1beta1 azureingressprohibitedtarget:v1" \
    --go-header-file ../code-generator/hack/boilerplate.go.txt

# go get github.com/knative/pkg/apis/istio/v1alpha3

# go mod vendor

echo -e "Generate Istio CRDs..."
../code-generator/generate-groups.sh \
    all \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client \
    github.com/knative/pkg/apis "istio:v1alpha3" \
    --go-header-file ../code-generator/hack/boilerplate.go.txt