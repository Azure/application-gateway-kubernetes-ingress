#!/bin/bash
set -ex

. helper.sh

# install
InstallAGIC

# set up shared backend
SetupSharedBackend

# run test
go mod init || true
go test -v -timeout 120m -tags e2e ./... >testoutput.txt || true
mv ./cmd/runner/report.xml report.e2e.xml

# install with custom tag
InstallAGIC "custom-ingress-class"

go test -v -timeout 120m -tags e2eingressclass ./... || true
mv ./cmd/runner/report.xml report.e2eingressclass.xml

# print test logs
cat testoutput.txt

EvaluateTestStatus
