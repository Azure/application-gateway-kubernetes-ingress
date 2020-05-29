#!/bin/bash
set -ex

function SetupRoleAssignments() {
    [[ -z "${APPGW_SUBSCRIPTION_ID}" ]] && (echo "APPGW_SUBSCRIPTION_ID is not set"; exit 1)
    [[ -z "${APPGW_RESOURCE_GROUP}" ]] && (echo "APPGW_RESOURCE_GROUP is not set"; exit 1)
    [[ -z "${APPGW_RESOURCE_ID}" ]] && (echo "APPGW_RESOURCE_ID is not set"; exit 1)
    [[ -z "${IDENTITY_CLIENT_ID}" ]] && (echo "IDENTITY_CLIENT_ID is not set"; exit 1)

    echo "Creating reader role assignment for AGIC identity"
    # az role assignment delete --role Reader --scope /subscriptions/${APPGW_SUBSCRIPTION_ID}/resourceGroups/${APPGW_RESOURCE_GROUP} --assignee ${IDENTITY_CLIENT_ID}
    az role assignment create --role Reader --scope /subscriptions/${APPGW_SUBSCRIPTION_ID}/resourceGroups/${APPGW_RESOURCE_GROUP} --assignee ${IDENTITY_CLIENT_ID}

    echo "Creating contributor role assignment for AGIC identity"
    # az role assignment delete --role Contributor --scope ${APPGW_RESOURCE_ID} --assignee ${IDENTITY_CLIENT_ID}
    az role assignment create --role Contributor --scope ${APPGW_RESOURCE_ID} --assignee ${IDENTITY_CLIENT_ID}
}

function DeleteOtherAGICVersions() {
    [[ -z "${VERSION}" ]] && (echo "VERSION is not set"; exit 1)

    list=$(helm ls --all --short -n agic | grep -v agic-${VERSION})
    if [[ $list != "" ]]
    then
        helm delete $list -n agic
    fi
}

function InstallAGIC() {
    [[ -z "${VERSION}" ]] && (echo "VERSION is not set"; exit 1)
    [[ -z "${APPGW_RESOURCE_ID}" ]] && (echo "APPGW_RESOURCE_ID is not set"; exit 1)
    [[ -z "${IDENTITY_RESOURCE_ID}" ]] && (echo "IDENTITY_RESOURCE_ID is not set"; exit 1)
    [[ -z "${IDENTITY_CLIENT_ID}" ]] && (echo "IDENTITY_CLIENT_ID is not set"; exit 1)

    DeleteOtherAGICVersions || true

    kubectl create namespace agic || true

    echo "Installing BuildId ${version}"

    helm repo add staging https://appgwingress.blob.core.windows.net/ingress-azure-helm-package-staging/
    helm repo update

    # AAD pod identity is taking time to assign identity. Timeout is set to 60 sec
    helm upgrade --install agic-${VERSION} staging/ingress-azure \
    --set appgw.applicationGatewayID=${APPGW_RESOURCE_ID} \
    --set armAuth.type=aadPodIdentity \
    --set armAuth.identityResourceID=${IDENTITY_RESOURCE_ID} \
    --set armAuth.identityClientID=${IDENTITY_CLIENT_ID} \
    --set rbac.enabled=true \
    --timeout 120s \
    --wait \
    -n agic \
    --version ${VERSION}
}

# Setup role assignments in case they got deleted
SetupRoleAssignments

# install
InstallAGIC

# run test
go mod init || true
go test -v -timeout 60m -tags e2e ./... > testoutput.txt || { echo "go test returned non-zero"; cat testoutput.txt; exit 1; }
