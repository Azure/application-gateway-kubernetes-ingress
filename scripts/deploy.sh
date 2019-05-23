#!/bin/bash
echo "Logging into docker...."
echo "$DOCKER_PASSWORD" | docker login "$ACR_REGISTRY" -u "$DOCKER_USERNAME" --password-stdin
echo "Login successful. Getting ready to deploy."

set -x
pushd "$TRAVIS_BUILD_DIR/build"
cmake --build . --target dockerpush
popd
set +x
