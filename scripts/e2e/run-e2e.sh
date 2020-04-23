#!/bin/bash
set -x

[[ -z "${version}" ]] && (echo "version is not set"; exit 1)
[[ -z "${applicationGatewayId}" ]] && (echo "buiapplicationGatewayIdldid is not set"; exit 1)
[[ -z "${identityResourceId}" ]] && (echo "identityResourceId is not set"; exit 1)
[[ -z "${identityClientId}" ]] && (echo "identityClientId is not set"; exit 1)

function InstallAGIC() {
    [[ -z "${version}" ]] && (echo "version is not set"; exit 1)
    [[ -z "${applicationGatewayId}" ]] && (echo "buiapplicationGatewayIdldid is not set"; exit 1)
    [[ -z "${identityResourceId}" ]] && (echo "identityResourceId is not set"; exit 1)
    [[ -z "${identityClientId}" ]] && (echo "identityClientId is not set"; exit 1)

    echo "Installing BuildId ${version}"

    list=$(helm ls --all --short)
    if [[ $list != "" ]]
    then
        helm delete $list
    fi

    helm repo add staging https://appgwingress.blob.core.windows.net/ingress-azure-helm-package-staging/
    helm repo update

    helm upgrade --install agic-${version} staging/ingress-azure \
    --set appgw.applicationGatewayID=${applicationGatewayId} \
    --set armAuth.type=aadPodIdentity \
    --set armAuth.identityResourceID=${identityResourceId} \
    --set armAuth.identityClientID=${identityClientId} \
    --set rbac.enabled=true \
    -n agic \
    --version ${version}
}

# install
InstallAGIC
