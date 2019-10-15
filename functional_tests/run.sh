#!/usr/bin/env bash

set -auexo pipefail

go test -v $(go list ./... | grep 'application-gateway'); echo $?
