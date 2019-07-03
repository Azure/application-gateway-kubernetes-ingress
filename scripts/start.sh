#!/bin/bash

set -aueo pipefail

source .env

# colors
COLOR_RESET='\e[0m'
COLOR_BLUE='\e[44;97m'
COLOR_RED='\e[101;97m'
COLOR_GREEN='\e[42;97m'

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

# Feature Flags
export APPGW_ENABLE_SAVE_CONFIG_TO_FILE="true"

# Run
./bin/appgw-ingress \
    --in-cluster=false \
    --kubeconfig="$KUBE_CONFIG_FILE" \
    --apiserver-host="$AKS_API_SERVER" \
    --verbosity=5
