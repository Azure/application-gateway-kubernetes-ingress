#!/bin/bash

export GOOS=linux
export GOBIN=`pwd`/bin
GO_PKGS="./cmd/... ./pkg/..."
GO_PROJ="github.com/Azure/application-gateway-kubernetes-ingress"

echo -e "\e[44;97m Running go lint.. \e[0m"
golint $GO_PKGS > ./lint.out
cat ./lint.out 
if [ -s ./lint.out ]; then
    echo -e "\e[101;97m GOLINT FAILED \e[0m"``
    exit 1
else
    echo -e "\e[42;97m GOLINT SUCCESS \e[0m"
fi

echo -e "\e[44;97m Running go vet ... \e[0m"
if go vet -v -source $GO_PKGS; then
    echo -e "\e[42;97m GOVET SUCCESS \e[0m"
else
    echo -e "\e[101;97m GOVET FAILED \e[0m"``
    exit 1
fi

echo -e "\e[44;97m Compiling ... \e[0m"
if  go install -v ./cmd/appgw-ingress; then
    chmod -R 777 bin
    echo -e "\e[42;97m BUILD SUCCESS \e[0m"
else
    echo -e "\e[101;97m BUILD FAILED \e[0m"
    exit 1
fi