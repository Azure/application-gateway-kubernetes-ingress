#!/bin/bash
set -ex

function DeleteOtherAGICVersions() {
    [[ -z "${version}" ]] && (echo "version is not set"; exit 1)

    list=$(helm ls --all --short -n agic | grep -v agic-${version})
    if [[ $list != "" ]]
    then
        helm delete $list -n agic
    fi
}

function InstallAGIC() {
    [[ -z "${version}" ]] && (echo "version is not set"; exit 1)
    [[ -z "${applicationGatewayId}" ]] && (echo "buiapplicationGatewayIdldid is not set"; exit 1)
    [[ -z "${identityResourceId}" ]] && (echo "identityResourceId is not set"; exit 1)
    [[ -z "${identityClientId}" ]] && (echo "identityClientId is not set"; exit 1)

    DeleteOtherAGICVersions || true

    kubectl create namespace agic || true

    # clean up brownfield namespace if exist
    kubectl delete ns test-brownfield-ns || true

    echo "Installing BuildId ${version}"

    helm repo add staging https://appgwingress.blob.core.windows.net/ingress-azure-helm-package-staging/
    helm repo update

    # AAD pod identity is taking time to assign identity. Timeout is set to 120 sec
    helm upgrade --install agic-${version} staging/ingress-azure \
    --set appgw.applicationGatewayID=${applicationGatewayId} \
    --set armAuth.type=aadPodIdentity \
    --set armAuth.identityResourceID=${identityResourceId} \
    --set armAuth.identityClientID=${identityClientId} \
    --set rbac.enabled=true \
    --set appgw.shared=false \
    --timeout 120s \
    --wait \
    -n agic \
    --version ${version}

    # apply backends to test prohibited target, wait for 90s to apply appgw config
    kubectl apply -f test-prohibit-backend.yaml && sleep 90
}

function SetupSharedBackend() {
    # install agic with shared enabled
    helm upgrade --install agic-${version} staging/ingress-azure \
    --set appgw.applicationGatewayID=${applicationGatewayId} \
    --set armAuth.type=aadPodIdentity \
    --set armAuth.identityResourceID=${identityResourceId} \
    --set armAuth.identityClientID=${identityClientId} \
    --set rbac.enabled=true \
    --set appgw.shared=true \
    --timeout 120s \
    --wait \
    -n agic \
    --version ${version}

    # apply customized prohibited policy for blacklisted backend
    kubectl apply -f prohibit-blacklist-service.yaml

    # get all the prohibited target config
    kubectl get AzureIngressProhibitedTargets -n agic -o yaml 

    # delete default prohibit-all-targets
    # blacklist-service shall be kept after porhibited policy applied
    kubectl delete AzureIngressProhibitedTarget prohibit-all-targets -n agic
}

# install
InstallAGIC

# set up shared backend
SetupSharedBackend

# run test
go mod init || true
go test -v -timeout 60m -tags e2e ./... > testoutput.txt || { echo "go test returned non-zero"; cat testoutput.txt; exit 1; }