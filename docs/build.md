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
In the `scripts` directory you will find `start.sh`. This script builds and runs the ingress controller on your development machine.
To successfully start the ingress controller via `./scripts/start.sh` you need to create the following files:
  - `~/.azure/azureAuth.json` - Use "az ad create-for-rbac --sdk-auth" command to [create these credentials](https://docs.microsoft.com/en-us/dotnet/api/overview/azure/containerinstance?view=azure-dotnet#authentication)
  - `~/.azure/subscription` - Place the subscription UUID of your AKS cluster on a single line
  - `~/.azure/resource-group` - Save the AKS Resource Group name on a single line
  - `~/.azure/app-gateway` - Place the Application Gateway name on a single line

Fill-in the values for your AKS cluster's **subscription**, **resource group**, **application gateway name**, and **AKS API server address**.
The script will create a `.build/` directory, compile and install the binary in it, then run the application.

### Run
With `$HOME/.azure/azureAuth.json` and `.dev/start.sh` created, you are ready to start the K8s ingress on your workstation. Execute `source .dev/start.sh` from within the root directory of the repo. The Ingress will connect to AKS, gather details on your running pods and configure the given Application Gateway.
