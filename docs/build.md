## Building

This is a CMake-based project. Build targets include:

- `ALL_BUILD` (default target) builds `appgw-ingress` and `dockerize` target
- `devenv` builds a docker image with configured development environment
- `vendor` installs dependency using `go mod` in a docker container with image from `devenv` target
- `appgw-ingress` builds the binary for this controller in a docker container with image from `devenv` target
- `dockerize` builds a docker image with the binary from `appgw-ingress` target
- `dockerpush` pushes the docker image to a container registry with prefix defined in CMake variable `<deployment_push_prefix>`

To run the CMake targets:

1. `mkdir build && cd build` creates and enters a build directory
2. `cmake ..` generates project configuration in the build directory
3. `cmake --build .` to build the default target,
    or `cmake --build . --target <target_name>` to specify a target to run from above
