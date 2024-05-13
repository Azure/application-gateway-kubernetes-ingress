function DeleteOtherAGICVersions() {
    [[ -z "${version}" ]] && (
        echo "version is not set"
        exit 1
    )

    list=$(helm ls --all --short -n agic | grep -v agic-${version})
    if [[ $list != "" ]]; then
        helm delete $list -n agic
    fi
}

function InstallAGIC() {
    [[ -z "${version}" ]] && (
        echo "version is not set"
        exit 1
    )

   [[ -z "${applicationGatewayId}" ]] && (
        echo "applicationGatewayId is not set"
        exit 1
    )
    [[ -z "${identityResourceId}" ]] && (
        echo "identityResourceId is not set"
        exit 1
    )
    [[ -z "${identityClientId}" ]] && (
        echo "identityClientId is not set"
        exit 1
    )

    DeleteOtherAGICVersions || true

    kubectl create namespace agic || true

    # clean up brownfield namespace if exist
    kubectl delete ns test-brownfield-ns || true

    echo "Installing BuildId ${version}"

    helm repo add staging https://appgwingress.blob.core.windows.net/ingress-azure-helm-package-staging/
    helm repo update

    # AAD pod identity is taking time to assign identity. Timeout is set to 120 sec
    helm upgrade --install agic-${version} staging/ingress-azure \
        -f ./helm-config-with-prohibited-rules.yaml \
        --set appgw.applicationGatewayID=${applicationGatewayId} \
        --set armAuth.type=workloadIdentity \
        --set armAuth.identityClientID=${identityClientId} \
        --set kubernetes.ingressClass="$1" \
        --timeout 120s \
        --wait \
        -n agic \
        --version ${version}
}

function SetupApplicationGateway() {
    [[ -z "${version}" ]] && (
        echo "version is not set"
        exit 1
    )
    [[ -z "${applicationGatewayId}" ]] && (
        echo "applicationGatewayId is not set"
        exit 1
    )
    [[ -z "${identityResourceId}" ]] && (
        echo "identityResourceId is not set"
        exit 1
    )
    [[ -z "${identityClientId}" ]] && (
        echo "identityClientId is not set"
        exit 1
    )

    gatewayName=$(echo $applicationGatewayId | cut -d'/' -f9)
    groupName=$(echo $applicationGatewayId | cut -d'/' -f5)

    az network application-gateway probe create \
        --gateway-name $gatewayName \
        --resource-group $groupName \
        --name msProbe \
        --path / \
        --protocol Https \
        --host www.microsoft.com \
        --interval 30 \
        --timeout 30 \
        --threshold 3

    az network application-gateway http-settings create \
        --gateway-name $gatewayName \
        --resource-group $groupName \
        --name msSettings \
        --port 443 \
        --protocol Https \
        --cookie-based-affinity Disabled \
        --timeout 30 \
        --probe msProbe \
        --path "/"

    az network application-gateway address-pool create \
        --gateway-name $gatewayName \
        --resource-group $groupName \
        --name msPool \
        --servers www.microsoft.com
    
    az network application-gateway address-pool create \
        --gateway-name $gatewayName \
        --resource-group $groupName \
        --name msEmpty
    
    az network application-gateway url-path-map create \
        --gateway-name $gatewayName \
        --resource-group $groupName \
        --name msPathMap \
        --rule-name msPathRule \
        --paths "/landing/*" \
        --address-pool msPool \
        --default-address-pool msEmpty \
        --http-settings msSettings \
        --default-http-settings msSettings

    az network application-gateway http-listener create \
        --gateway-name $gatewayName \
        --resource-group $groupName \
        --name msListener \
        --frontend-ip appGatewayFrontendIP \
        --frontend-port fake-prod-fp-80 \
        --host-names www.microsoft.com \

    az network application-gateway rule create \
        --gateway-name $gatewayName \
        --resource-group $groupName \
        --name msRule \
        --http-settings msSettings \
        --address-pool msPool \
        --http-listener msListener \
        --rule-type  PathBasedRouting \
        --url-path-map msPathMap \
        --priority 1
}

function EvaluateTestStatus() {
    failedCount=$(grep "type=\"Failure\"" report.*.xml | wc -l | cut -d' ' -f8)
    if [[ failedCount -ne 0 ]]; then
        exit 1
    fi

    exit 0
}
