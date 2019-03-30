cmake_minimum_required(VERSION 2.8.9)
project(application-gateway-kubernetes-ingress)
string(TOLOWER ${CMAKE_PROJECT_NAME} project_name_lowercase)

set(package_name "github.com/Azure/application-gateway-kubernetes-ingress")
set(devenv_image_name "${project_name_lowercase}-dev")
set(devenv_workpath "/gopath/src/${package_name}")

set(deployment_image_name ${project_name_lowercase}
    CACHE STRING "string <deployment_image_name> for docker tag/push <prefix>/<deployment_image_name>")
set(deployment_push_prefix "" 
    CACHE STRING "string <prefix> for docker tag/push <prefix>/<deployment_image_name>")
set(deployment_image_version "latest"
    CACHE STRING "version of the deployment image")

message("[image name] ${deployment_push_prefix}/${deployment_image_name}:${deployment_image_version}")

add_custom_command(
    OUTPUT  ${CMAKE_SOURCE_DIR}/vendor/vendor.stamp
    COMMAND docker run --rm --security-opt seccomp:unconfined  -t
            -v ${CMAKE_SOURCE_DIR}:${devenv_workpath}
            -w ${devenv_workpath}
            --name ${devenv_image_name} ${devenv_image_name}
            /bin/bash -c './scripts/vendor.sh'
    COMMAND cmake -E touch ${CMAKE_SOURCE_DIR}/vendor/vendor.stamp
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
    DEPENDS glide.yaml devenv
    COMMENT "running glide install -v"
)

# Go vendoring
add_custom_target(
    vendor
    DEPENDS ${CMAKE_SOURCE_DIR}/vendor/vendor.stamp
)

# Docker image for build environment
add_custom_target(devenv
    COMMAND docker build -t ${devenv_image_name}
            -f dockerfiles/devenv.Dockerfile ${CMAKE_SOURCE_DIR}
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
    COMMENT "building devenv docker image"
)

add_custom_command(
    OUTPUT .always_rebuild
    COMMAND cmake -E echo
)

# Custom command to generate the GoLang binaries
add_custom_command(
    OUTPUT ${CMAKE_SOURCE_DIR}/bin/appgw-ingress
    COMMAND docker run --rm --security-opt seccomp:unconfined  -t
            -v ${CMAKE_SOURCE_DIR}:${devenv_workpath}
            -w ${devenv_workpath}
            --name ${devenv_image_name}-devenv ${devenv_image_name}
            /bin/bash -c './scripts/build.sh'
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
    DEPENDS devenv .always_rebuild
    COMMENT "compiling project in devenv"
)

# Generate a new helm release.
add_custom_target(helm-release
    COMMAND docker run --rm --security-opt seccomp:unconfined  -t
    -v ${CMAKE_SOURCE_DIR}:${devenv_workpath}
    -w ${devenv_workpath}/helm
    --name ${devenv_image_name}-devenv ${devenv_image_name}
    /bin/bash -c './release.sh'
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
)

# Build target for building the GoLang binaries
add_custom_target(
    appgw-ingress
    DEPENDS ${CMAKE_SOURCE_DIR}/bin/appgw-ingress
)

# Test target for pkg/utils
add_custom_target(test-pkg-utils
    COMMAND docker run --rm --security-opt seccomp:unconfined  -t
            -v ${CMAKE_SOURCE_DIR}:${devenv_workpath}
            -w ${devenv_workpath}
            --name ${devenv_image_name}-devenv ${devenv_image_name}
            ginkgo -v pkg/utils
    DEPENDS appgw-ingress
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
)

# Test target for pkg/k8scontext
add_custom_target(test-pkg-k8scontext
    COMMAND docker run --rm --security-opt seccomp:unconfined  -t
            -v ${CMAKE_SOURCE_DIR}:${devenv_workpath}
            -w ${devenv_workpath}
            --name ${devenv_image_name}-devenv ${devenv_image_name}
            ginkgo -v pkg/k8scontext
    DEPENDS appgw-ingress
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
)

# Test target for pkg/appgw
add_custom_target(test-pkg-appgw
    COMMAND docker run --rm --security-opt seccomp:unconfined  -t
            -v ${CMAKE_SOURCE_DIR}:${devenv_workpath}
            -w ${devenv_workpath}
            --name ${devenv_image_name}-devenv ${devenv_image_name}
            ginkgo -v pkg/appgw
    DEPENDS appgw-ingress
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
)

# Top-level test target for running all go tests
add_custom_target(go-tests
    DEPENDS test-pkg-utils test-pkg-k8scontext test-pkg-appgw
)

# Target for building a docker image for deploying on a Kubernetes cluster.
add_custom_target(dockerize
    COMMAND docker build -t ${deployment_image_name}:${deployment_image_version}
            -f dockerfiles/deploy.Dockerfile ${CMAKE_SOURCE_DIR}
    WORKING_DIRECTORY ${CMAKE_SOURCE_DIR}
    DEPENDS go-tests
    COMMENT "building deployment docker image"
)

# Target for pushing the docker image of the Kuberentes pod to a container registry.
add_custom_target(dockerpush
    COMMAND docker tag ${deployment_image_name}:${deployment_image_version} ${deployment_push_prefix}/${deployment_image_name}:${deployment_image_version}
    COMMAND docker tag ${deployment_image_name}:${deployment_image_version} ${deployment_push_prefix}/${deployment_image_name}:latest
    COMMAND docker push ${deployment_push_prefix}/${deployment_image_name}:${deployment_image_version}
    COMMAND docker push ${deployment_push_prefix}/${deployment_image_name}:latest
    DEPENDS dockerize
    COMMENT "pushing to container registry"
)

# Target to capture ALL semantics.
add_custom_target(appgw-ingress-all
    ALL
    DEPENDS appgw-ingress go-tests
)