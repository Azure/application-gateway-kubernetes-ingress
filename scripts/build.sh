#!/bin/bash
echo -e "\e[44;97m Compiling ... \e[0m"
if GOOS=linux GOBIN=`pwd`/bin go install -v ./cmd/appgw-ingress; then
    chmod -R 777 bin
    echo -e "\e[42;97m BUILD SUCCESS \e[0m"
else
    echo -e "\e[101;97m BUILD FAILED \e[0m"
    exit 1
fi

echo -e "\e[44;97m Running go lint ... \e[0m"
if GOOS=linux GOBIN=`pwd`/bin golint ./cmd/... ./pkg/...; then
    chmod -R 777 bin
    echo -e "\e[42;97m GOLINT SUCCESS \e[0m"
else
    echo -e "\e[101;97m GOLINT FAILED \e[0m"
    exit 1
fi

echo -e "\e[44;97m Running go vet ... \e[0m"
if GOOS=linux GOBIN=`pwd`/bin go vet -v -source ./cmd/... ./pkg/...; then
    chmod -R 777 bin
    echo -e "\e[42;97m GOVET SUCCESS \e[0m"
else
    echo -e "\e[101;97m GOVET FAILED \e[0m"
    exit 1
fi