#!/bin/bash
set -ex

. helper.sh

# install
InstallAGIC

# set up shared backend
SetupSharedBackend

# run test
go mod init || true
go test -v -timeout 60m -tags e2e ./... >testoutput.txt || {
    echo "go test returned non-zero"
    cat testoutput.txt
    exit 1
}
