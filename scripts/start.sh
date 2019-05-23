#!/bin/bash

set -aueo pipefail

# colors
COLOR_RESET='\e[0m'
COLOR_BLUE='\e[44;97m'
COLOR_RED='\e[101;97m'
COLOR_GREEN='\e[42;97m'

# Create the following files:
#   ~/.azure/azureAuth.json --> Use "az ad create-for-rbac --sdk-auth" command to create these credentials. See https://docs.microsoft.com/en-us/dotnet/api/overview/azure/containerinstance?view=azure-dotnet#authentication
#   ~/.azure/subscription --> Place the subscription UUID on a single line
#   ~/.azure/resource-group --> Place the Resource Group name on a single line
#   ~/.azure/app-gateway --> Place the Application Gateway name on a single line

readonly AZURE_AUTH_LOCATION="$HOME/.azure/azureAuth.json"
if [ ! -f "$AZURE_AUTH_LOCATION" ]; then
  echo "File $AZURE_AUTH_LOCATION not found! You need Azure credentials for the ingress to be able to control your App Gateway."
  exit 1
fi

AZURE_SUBSCRIPTION_FILE="$HOME/.azure/subscription"
if [ ! -f "$AZURE_SUBSCRIPTION_FILE" ]; then
  echo "File $AZURE_SUBSCRIPTION_FILE not found! You need to save the Azure subscription ID in it."
  exit 1
fi

AZURE_RG_FILE="$HOME/.azure/resource-group"
if [ ! -f "$AZURE_RG_FILE" ]; then
  echo "File $AZURE_RG_FILE not found! You need to save the Azure resource group in it."
  exit 1
fi

AZURE_AG_FILE="$HOME/.azure/app-gateway"
if [ ! -f "$AZURE_AG_FILE" ]; then
  echo "File $AZURE_AG_FILE not found! You need to save the Azure Application Gateway name in it."
  exit 1
fi

AKS_API_SERVER_FILE="$HOME/.azure/aks-api-server"
if [ ! -f "$AKS_API_SERVER_FILE" ]; then
  echo "File $AKS_API_SERVER_FILE not found! You need to save the AKS API server address in it."
  exit 1
fi

KUBE_CONFIG_FILE="$HOME/.kube/config"
if [ ! -f "$KUBE_CONFIG_FILE" ]; then
  echo "File $KUBE_CONFIG_FILE not found! You need Kubernetes credentials saved in it."
  exit 1
fi

readonly APPGW_SUBSCRIPTION_ID=$(cat "$AZURE_SUBSCRIPTION_FILE")
readonly APPGW_RESOURCE_GROUP=$(cat "$AZURE_RG_FILE")
readonly APPGW_NAME=$(cat "$AZURE_AG_FILE")
readonly KUBERNETES_WATCHNAMESPACE=default
AKS_API_SERVER=$(cat "$AKS_API_SERVER_FILE")

# The variables below will be used by the appgw-ingress binary
export AZURE_AUTH_LOCATION
export APPGW_SUBSCRIPTION_ID
export APPGW_RESOURCE_GROUP
export APPGW_NAME
export KUBERNETES_WATCHNAMESPACE


GOBIN=$(pwd)/bin

echo -e "$COLOR_RED Cleanup: $COLOR_RESET delete $GOBIN"
rm -rf "$GOBIN"

ORG_PATH="github.com/Azure"
PROJECT_NAME="application-gateway-kubernetes-ingress"
REPO_PATH="${ORG_PATH}/${PROJECT_NAME}"
VERSION_VAR="${REPO_PATH}/pkg/version.Version"
DATE_VAR="${REPO_PATH}/pkg/version.BuildDate"
COMMIT_VAR="${REPO_PATH}/pkg/version.GitCommit"
VERSION=$(git describe --abbrev=0 --tags)
BUILD_DATE=$(date +%Y-%m-%d-%H:%MT%z)
GIT_HASH=$(git rev-parse --short HEAD)

echo -e "$COLOR_BLUE Compiling ... $COLOR_RESET"
GOOS=linux go install -ldflags "-s -X ${VERSION_VAR}=${VERSION} -X ${DATE_VAR}=${BUILD_DATE} -X ${COMMIT_VAR}=${GIT_HASH}" -v ./cmd/appgw-ingress
RESULT=$?
if [ "$RESULT" -eq "0" ]; then
    chmod -R 777 bin
    echo -e "$COLOR_GREEN Build SUCCEEDED $COLOR_RESET"
else
    echo -e "$COLOR_RED Build FAILED $COLOR_RESET"
    exit 1
fi

# Print Version
./bin/appgw-ingress --version || true

# Run
./bin/appgw-ingress \
    --in-cluster=false \
    --kubeconfig="$KUBE_CONFIG_FILE" \
    --apiserver-host="$AKS_API_SERVER"
