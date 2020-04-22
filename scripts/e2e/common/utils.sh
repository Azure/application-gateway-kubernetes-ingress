function logError () {
    printf "[$(date +'%D %T') Bootstrap]  $1\n" >&2
}

function CleanUp() {
	namspaces=$(kubectl get -o custom-columns=:metadata.name ns | grep -v "default\|kube\|agic")
    if [[ $namspaces != "" ]]
    then
        echo $namspaces | xargs kubectl delete namespace
    fi
}

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

function GetIngressIP() {
    for ((i=1;i<=100;i++));
    do
        publicIP=$(kubectl get ingress/$1 -n $2 -o json | jq -r ".status.loadBalancer.ingress[0].ip")
        if [[ $publicIP != "null" && $publicIP != "" ]]
        then
	    echo $publicIP
            return
        fi
        echo "Public Ip is null. Will retry again in 1 sec"
        sleep 1
    done
}


function InvokeCurl() {
    statusCode=""
    rawOutput=""
    response=""
    
    for  ((i=1;i<=100;i++));
    do
        if [ $# -eq 2 ]
        then
            rawOutput=$(curl -sS $1 -w "BOOTSTRAP_HTTP_STATUS_CODE:%{http_code}")
        elif [ $# -eq 3 ]
        then
            rawOutput=$(curl -sS $1 -H "$3" -w "BOOTSTRAP_HTTP_STATUS_CODE:%{http_code}")
        elif [ $# -eq 4 ]
        then
            rawOutput=$(curl -sS $1 -H "$3" $4 -w "BOOTSTRAP_HTTP_STATUS_CODE:%{http_code}")
        else
            logError "InvokeCurl invoked with invalid number of arguments $#"
        fi
        
        statusCode=$(echo $rawOutput | grep -oh "BOOTSTRAP_HTTP_STATUS_CODE:.*")
        _curlContent=$(echo $rawOutput | sed -n 's/BOOTSTRAP_HTTP_STATUS_CODE:.*//p')

        # Extract value of status code
        IFS=':' read -ra response <<< "$statusCode"
        statusCode=${response[1]}
        if [[ $statusCode != $2 ]]
        then
            logError "Curl call failed with following status code $statusCode. Content: '$_curlContent'"
        else
            break
        fi

        echo "Sleeping for 5 second..."
        sleep 5
    done
}
