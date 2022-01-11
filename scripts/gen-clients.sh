#!/bin/bash

set -auexo pipefail

# This script requres a checkout of https://github.com/kubernetes/code-generator in ../
# For more information read https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/
# To generate crds, run this in the base directory of AGIC repo. Generated files will be in ~/go/src/ dir

# echo -e "Cleanup previously generated code..."
# rm -rf pkg/client $(find ./pkg -name 'zz_*.go')

echo -e "Generate Application Gateway CRDs..."
../code-generator/generate-groups.sh \
    all \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis \
    "azureapplicationgatewayinstanceupdatestatus:v1beta1 azureapplicationgatewaybackendpool:v1beta1 azureingressprohibitedtarget:v1 loaddistributionpolicy:v1beta1 azureapplicationgatewayheaderrewrite:v1beta1" \
    --go-header-file ../code-generator/hack/boilerplate.go.txt

echo -e "Generate Azure Multi-Cluster CRDs..."
../code-generator/generate-groups.sh \
    all \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/azure_multicluster_crd_client \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis "multiclusterservice:v1alpha1 multiclusteringress:v1alpha1" \
    --go-header-file ../code-generator/hack/boilerplate.go.txt \
    -v 10

# go get github.com/knative/pkg/apis/istio/v1alpha3

# go mod vendor

echo -e "Generate Istio CRDs..."
../code-generator/generate-groups.sh \
    all \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/istio_crd_client \
    github.com/knative/pkg/apis "istio:v1alpha3" \
    --go-header-file ../code-generator/hack/boilerplate.go.txt