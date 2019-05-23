#!/bin/bash

# bash colors
COLOR_RESET='\e[0m'
COLOR_BLUE='\e[44;97m'
COLOR_RED='\e[101;97m'
COLOR_GREEN='\e[42;97m'

readonly GOOS=linux
export GOOS

readonly GOBIN=$(pwd)/bin
export GOBIN

GO_PROJ="github.com/Azure/application-gateway-kubernetes-ingress"
GO_PKGS=$(go list ./... | grep -v vendor/)
GO_FILES=$(find . -type f -name '*.go' -not -path "./vendor/*")

ORG_PATH="github.com/Azure"
PROJECT_NAME="application-gateway-kubernetes-ingress"
REPO_PATH="${ORG_PATH}/${PROJECT_NAME}"

VERSION_VAR="${REPO_PATH}/pkg/version.Version"
VERSION=$(git describe --abbrev=0 --tags)

DATE_VAR="${REPO_PATH}/pkg/version.BuildDate"
BUILD_DATE=$(date +%Y-%m-%d-%H:%MT%z)

COMMIT_VAR="${REPO_PATH}/pkg/version.GitCommit"
GIT_HASH=$(git rev-parse --short HEAD)

echo -e "$COLOR_BLUE Running go lint.. $COLOR_RESET"
golint "$GO_PKGS" > /tmp/lint.out

cat /tmp/lint.out
if [ -s /tmp/lint.out ]; then
    echo -e "$COLOR_RED golint FAILED $COLOR_RESET"
    exit 1
else
    echo -e "$COLOR_GREEN golint SUCCEEDED $COLOR_RESET"
fi

echo -e "$COLOR_BLUE Running govet ... $COLOR_RESET"
if go vet -v "$GO_PKGS"; then
    echo -e "$COLOR_GREEN govet SUCCEEDED $COLOR_RESET"
else
    echo -e "$COLOR_RED govet FAILED $COLOR_RESET"
    exit 1
fi

echo -e "$COLOR_BLUE Running goimports ... $COLOR_RESET"
goimports -local "$GO_PROJ" -w "$GO_FILES" > /tmp/goimports.out
cat /tmp/goimports.out
if [ -s /tmp/goimports.out ]; then
    echo -e "$COLOR_RED goimports FAILED $COLOR_RESET"
    exit 1
else
    echo -e "$COLOR_GREEN goimports SUCCEEDED $COLOR_RESET"
fi

echo -e "$COLOR_BLUE Compiling ... $COLOR_RESET"
if  go install -ldflags "-s -X ${VERSION_VAR}=${VERSION} -X ${DATE_VAR}=${BUILD_DATE} -X ${COMMIT_VAR}=${GIT_HASH}" -v ./cmd/appgw-ingress; then
    chmod -R 777 bin
    echo -e "$COLOR_GREEN Build SUCCEEDED $COLOR_RESET"
else
    echo -e "$COLOR_RED Build FAILED $COLOR_RESET"
    exit 1
fi
