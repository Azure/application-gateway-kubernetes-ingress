# Building the controller

## CMake options

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

## Running it locally

This section outlines the environment variables and files necessary to successfully compile and run the Go binary, then connect it to an [Azure Kubernetes Service](https://docs.microsoft.com/en-us/azure/aks/intro-kubernetes).

### Obtain Azure Credentials

In order to run the Go binary locally and control a remote AKS server, you need Azure credentials. These will be stored in a JSON file in your home directory.

Follow [these instructions](https://docs.microsoft.com/en-us/dotnet/api/overview/azure/containerinstance?view=azure-dotnet#authentication) to create the `$HOME/.azure/azureAuth.json` file. The file is generated via:

```bash
az ad sp create-for-rbac --subscription <your-azure-subscription-id> --sdk-auth > $HOME/.azure/azureAuth.json
```

The file will contain a JSON blob with the following shape:

```json
{
  "clientId": "...",
  "clientSecret": "...",
  "subscriptionId": "<your-azure-resource-group>",
  "tenantId": "...",
  "activeDirectoryEndpointUrl": "https://login.microsoftonline.com",
  "resourceManagerEndpointUrl": "https://management.azure.com/",
  "activeDirectoryGraphResourceId": "https://graph.windows.net/",
  "sqlManagementEndpointUrl": "https://management.core.windows.net:8443/",
  "galleryEndpointUrl": "https://gallery.azure.com/",
  "managementEndpointUrl": "https://management.core.windows.net/"
}
```

### Startup Script

In the `scripts` directory you will find `start.sh`. This script builds and runs the ingress controller on your local machine and connects to a remote AKS cluster. A `.env` file in the root of the repository is required.

Steps to run ingress controller:

1. Configure: `cp .env.example .env` and modify the environment variables in `.env` to match your config
1. Run: `./scripts/start.sh`
