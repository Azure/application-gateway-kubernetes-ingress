#!/bin/bash
set -x

[[ -z "${version}" ]] && (echo "version is not set"; exit 1)
[[ -z "${applicationGatewayId}" ]] && (echo "buiapplicationGatewayIdldid is not set"; exit 1)
[[ -z "${identityResourceId}" ]] && (echo "identityResourceId is not set"; exit 1)
[[ -z "${identityClientId}" ]] && (echo "identityClientId is not set"; exit 1)

source ./common/utils.sh

# install
InstallAGIC

# clean
CleanUp

# run test serially
test_scripts=$(find . -name run.sh)
for script in $test_scripts
do
    chmod +x $script

    # run the test
    eval $script

    # clean
    CleanUp
done
