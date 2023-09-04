#!/bin/bash

set -auexo pipefail

# This script requres a checkout of https://github.com/kubernetes/code-generator release-1.21 in ../
# For more information read https://blog.openshift.com/kubernetes-deep-dive-code-generation-customresources/
# To generate CRDs, run this in the base directory of AGIC repo. Generated files will be in ~/go/src/ dir. Copy them over to ./pkg folder.

# Commands to copy:
# cp ~/go/src/github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis           ./pkg -r
# cp ~/go/src/github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client     ./pkg -r

# echo -e "Cleanup previously generated code..."
# rm -rf pkg/client $(find ./pkg -name 'zz_*.go')

echo -e "Generate Application Gateway CRDs..."
../code-generator/generate-groups.sh \
    all \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/crd_client/agic_crd_client \
    github.com/Azure/application-gateway-kubernetes-ingress/pkg/apis \
    "azureapplicationgatewayinstanceupdatestatus:v1beta1 azureapplicationgatewaybackendpool:v1beta1 azureingressprohibitedtarget:v1 loaddistributionpolicy:v1beta1 azureapplicationgatewayrewrite:v1beta1" \
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