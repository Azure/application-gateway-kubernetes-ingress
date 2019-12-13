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

az extension add --name aks-preview
az extension update --name aks-preview

clusterName=$(kubectl config current-context)

read -p 'ResourceGroup for '$clusterName' cluster: ' resourceGroupName
read -p 'StorageAccountName: ' storageAccountName
read -p 'StorageAccountSAS: ' sasKey

file="/tmp/t0923"
kubectl get pods -l "app=ingress-azure" --template '{{range .items}}{{.metadata.namespace}}{{"/"}}{{.metadata.name}}{{"\n"}}{{end}}' > $file

## Construct the pods to get the logs from
while IFS= read -r line
do
  echo "$line"
  containers="$containers $line"
done < "$file"

echo "Collect Logs for following AGIC containers: $containers"
az aks kollect -n $clusterName -g $resourceGroupName --storage-account $storageAccountName --storage-account $storageAccountName --sas-token $sasKey --container-logs $containers

rm -rf $file