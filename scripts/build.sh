#!/bin/bash
echo -e "\e[44;97m Compiling ... \e[0m"
if GOOS=linux GOBIN=`pwd`/bin go install -v ./cmd/appgw-ingress; then
    chmod -R 777 bin
    echo -e "\e[42;97m SUCCESS \e[0m"
else
    echo -e "\e[101;97m FAILED \e[0m"
    exit 1
fi
