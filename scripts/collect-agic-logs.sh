#!/bin/bash
#This script uses AKS Periscope to collect the AGIC logs. See https://github.com/Azure/aks-periscope for more details

if ! [ "which az" ]; then
  echo "Cannot find Azure CLI. Please see to install if : https://docs.microsoft.com/en-us/cli/azure/install-azure-cli?view=azure-cli-latest"
  exit 1
fi

if ! [ "which kubectl" ]; then
  echo "Cannot find kubectl. Please install it"
  exit 1
fi

read -p 'Please confirm azure-cli is updated to latest version (y/N):' cliVersionConfirmation
if [ $cliVersionConfirmation != "y" ]
then
  exit 1
fi

az extension add --name aks-preview
az extension update --name aks-preview

clusterName=$(kubectl config current-context)

read -p 'ResourceGroup for '$clusterName' cluster: ' resourceGroupName
read -p 'StorageAccountName: ' storageAccountName
read -p 'StorageAccountSAS: ' sasKey

file="/tmp/agict0923"
declare -a logContainer
kubectl get pods -l "app=ingress-azure" --template '{{range .items}}{{.spec.nodeName}}{{"/"}}{{.metadata.namespace}}{{"/"}}{{.metadata.name}}{{"\n"}}{{end}}' > $file

## Construct the pods to get the logs from
while IFS= read -r line
do
  #echo "$line"
  IFS='/' read -ra pod_info <<< "$line"
  podName="${pod_info[1]}/${pod_info[2]}"
  logLocation="${pod_info[0]}/collector/containerlogs/${pod_info[1]}_${pod_info[2]}"
  logContainer+=($logLocation)
  containers="$containers $podName"
done < "$file"

if [ -z "$containers" ]
then
  echo "No AGIC deployments found"
  exit 1
fi

echo "Collect Logs for following AGIC containers: $containers"
az aks kollect -n $clusterName -g $resourceGroupName --storage-account "$storageAccountName" --sas-token "$sasKey" --container-logs "$containers"

printf '\n\n %s\n\n' "------- Please collect logs at following paths under the latest timestamped container named '$clusterName' in storage '$storageAccountName' ---"
printf '%s\n' "${logContainer[@]}"

rm -rf $file
