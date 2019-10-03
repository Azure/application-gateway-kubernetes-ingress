#!/bin/sh

go test -v $(go list ./... | grep 'application-gateway'); echo $?
