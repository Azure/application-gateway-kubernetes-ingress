#!/usr/bin/env bash

set -auexo pipefail

go test -v -tags unittest $(go list ./... | grep 'application-gateway'); echo $?
