#!/bin/bash

export GOOS=linux
export GOBIN=`pwd`/bin
GO_PROJ="github.com/Azure/application-gateway-kubernetes-ingress"
GO_PKGS=`go list ./... | grep -v vendor/`
echo $GO_PKGS

echo -e "\e[44;97m Running go lint.. \e[0m"
golint $GO_PKGS > /tmp/lint.out
cat /tmp/lint.out 
if [ -s /tmp/lint.out ]; then
    echo -e "\e[101;97m golint FAILED \e[0m"``
    exit 1
else
    echo -e "\e[42;97m golint SUCCESS \e[0m"
fi

echo -e "\e[44;97m Running govet ... \e[0m"
if go vet -v $GO_PKGS; then
    echo -e "\e[42;97m govet SUCCESS \e[0m"
else
    echo -e "\e[101;97m govet FAILED \e[0m"``
    exit 1
fi

echo -e "\e[44;97m Compiling ... \e[0m"
if  go install -v ./cmd/appgw-ingress; then
    chmod -R 777 bin
    echo -e "\e[42;97m Build SUCCESS \e[0m"
else
    echo -e "\e[101;97m Build FAILED \e[0m"
    exit 1
fi