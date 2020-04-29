#!/bin/bash
set -ex

function InstallAGIC() {
    [[ -z "${version}" ]] && (echo "version is not set"; exit 1)
    [[ -z "${applicationGatewayId}" ]] && (echo "buiapplicationGatewayIdldid is not set"; exit 1)
    [[ -z "${identityResourceId}" ]] && (echo "identityResourceId is not set"; exit 1)
    [[ -z "${identityClientId}" ]] && (echo "identityClientId is not set"; exit 1)

    echo "Installing BuildId ${version}"

    list=$(helm ls --all --short -n agic | grep -v agic-${version})
    if [[ $list != "" ]]
    then
        helm delete $list -n agic
    fi

    helm repo add staging https://appgwingress.blob.core.windows.net/ingress-azure-helm-package-staging/
    helm repo update

    # AAD pod identity is taking time to assign identity. Timeout is set to 60 sec
    helm upgrade --install agic-${version} staging/ingress-azure \
    --set appgw.applicationGatewayID=${applicationGatewayId} \
    --set armAuth.type=aadPodIdentity \
    --set armAuth.identityResourceID=${identityResourceId} \
    --set armAuth.identityClientID=${identityClientId} \
    --set rbac.enabled=true \
    --timeout 60s \
    --wait \
    -n agic \
    --version ${version}
}

# install
InstallAGIC

# run test
go mod init
go test -v --tags e2e ./... > testoutput.txt || { echo "go test returned non-zero"; cat testoutput.txt; exit 1; }
