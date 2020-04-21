#!/bin/bash

source ./pipeline.env

# install
# ./install.sh 8513

# run test serially
test_scripts=$(find . -name run.sh)

for script in $test_scripts
do
    eval $script
done
