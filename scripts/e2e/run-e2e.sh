#!/bin/bash
set -x
source ./pipeline.env
source ./common/utils.sh

# install
InstallAGIC

# clean
CleanUp

# run test serially
test_scripts=$(find . -name run.sh)
for script in $test_scripts
do
    eval $script
    CleanUp
    # clean up namespaces
done
